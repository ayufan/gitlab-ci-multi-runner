package kubernetes

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"

	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

var (
	kubeClient *client.Client
)

type kubernetesOptions struct {
	Image    string   `json:"image"`
	Services []string `json:"services"`
}

type executor struct {
	executors.AbstractExecutor

	prepod       *api.Pod
	pod          *api.Pod
	options      *kubernetesOptions
	extraOptions Options
}

func (s *executor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	err := s.AbstractExecutor.Prepare(globalConfig, config, build)
	if err != nil {
		return err
	}

	if kubeClient == nil {
		kubeClient, err = getKubeClient(config.Kubernetes)
		if err != nil {
			return err
		}
	}

	if s.BuildScript.PassFile {
		return fmt.Errorf("Kubernetes doesn't support shells that require script file")
	}

	err = build.Options.Decode(&s.options)
	if err != nil {
		return err
	}

	s.extraOptions = DefaultOptions{s.Build.GetAllVariables()}

	if !s.Config.Kubernetes.AllowPrivileged && s.extraOptions.Privileged() {
		return fmt.Errorf("Runner does not allow privileged containers")
	}

	s.Println("Using Kubernetes executor with image", s.options.Image, "...")

	return nil
}

func (s *executor) Cleanup() {
	if s.pod != nil {
		err := kubeClient.Pods(s.pod.Namespace).Delete(s.pod.Name, nil)

		if err != nil {
			s.Errorln("Error cleaning up pod: %s", err.Error())
		}
	}
	s.AbstractExecutor.Cleanup()
}

func buildVariables(bv common.BuildVariables) []api.EnvVar {
	e := make([]api.EnvVar, len(bv))
	for i, b := range bv {
		e[i] = api.EnvVar{
			Name:  b.Key,
			Value: b.Value,
		}
	}
	return e
}

func (s *executor) buildContainer(name, image string, command ...string) api.Container {
	path := strings.Split(s.Shell.Build.BuildDir, "/")
	path = path[:len(path)-1]

	privileged := s.extraOptions.Privileged()

	return api.Container{
		Name:    name,
		Image:   image,
		Command: command,
		Env:     buildVariables(s.Build.GetAllVariables().PublicOrInternal()),
		VolumeMounts: []api.VolumeMount{
			api.VolumeMount{
				Name:      "repo",
				MountPath: strings.Join(path, "/"),
			},
		},
		SecurityContext: &api.SecurityContext{
			Privileged: &privileged,
		},
		Stdin: true,
	}
}

func (s *executor) Run(cmd common.ExecutorCommand) error {
	var err error
	s.Debugln("Starting Kubernetes command...")

	if s.pod == nil {
		services := make([]api.Container, len(s.options.Services))
		for i, image := range s.options.Services {
			services[i] = s.buildContainer(fmt.Sprintf("svc-%d", i), image)
		}

		s.pod, err = kubeClient.Pods(s.Config.Kubernetes.Namespace).Create(&api.Pod{
			ObjectMeta: api.ObjectMeta{
				GenerateName: s.Build.ProjectUniqueName(),
				Namespace:    s.Config.Kubernetes.Namespace,
			},
			Spec: api.PodSpec{
				Volumes: []api.Volume{
					api.Volume{
						Name: "repo",
						VolumeSource: api.VolumeSource{
							EmptyDir: &api.EmptyDirVolumeSource{},
						},
					},
				},
				RestartPolicy: api.RestartPolicyNever,
				Containers: append([]api.Container{
					s.buildContainer("build", s.options.Image, s.BuildScript.DockerCommand...),
					s.buildContainer("pre", "munnerz/gitlab-runner-helper", s.BuildScript.DockerCommand...),
				}, services...),
			},
		})

		if err != nil {
			return err
		}
	}

	errc := func() <-chan error {
		errc := make(chan error, 1)
		go func() {
			defer close(errc)

			status, err := waitForPodRunning(kubeClient, s.pod, s.BuildLog)

			if err != nil {
				errc <- err
				return
			}

			if status != api.PodRunning {
				errc <- fmt.Errorf("pod failed to enter running state: %s", status)
				return
			}

			config, err := getKubeClientConfig(s.Config.Kubernetes)

			if err != nil {
				errc <- err
				return
			}

			var containerName string
			switch {
			case cmd.Predefined:
				containerName = "pre"
			default:
				containerName = "build"
			}

			exec := ExecOptions{
				PodName:       s.pod.Name,
				Namespace:     s.pod.Namespace,
				ContainerName: containerName,
				Command:       s.BuildScript.DockerCommand,
				In:            strings.NewReader(cmd.Script),
				Out:           s.BuildLog,
				Err:           s.BuildLog,
				Stdin:         true,
				Config:        config,
				Client:        kubeClient,
				Executor:      &DefaultRemoteExecutor{},
			}

			errc <- exec.Run()
		}()

		return errc
	}()

	select {
	case err := <-errc:
		return err
	case _ = <-cmd.Abort:
		return fmt.Errorf("build aborted")
	}
}

func init() {
	options := executors.ExecutorOptions{
		SharedBuildsDir: false,
		Shell: common.ShellScriptInfo{
			Shell:         "bash",
			Type:          common.NormalShell,
			RunnerCommand: "/gitlab-runner-helper",
		},
		ShowHostname:     true,
		SupportedOptions: []string{"image", "services", "artifacts", "cache"},
	}

	creator := func() common.Executor {
		return &executor{
			AbstractExecutor: executors.AbstractExecutor{
				ExecutorOptions: options,
			},
		}
	}

	featuresUpdater := func(features *common.FeaturesInfo) {
		features.Variables = true
		features.Image = true
		features.Services = true
		features.Artifacts = true
		features.Cache = true
	}

	common.RegisterExecutor("kubernetes", executors.DefaultExecutorProvider{
		Creator:         creator,
		FeaturesUpdater: featuresUpdater,
	})
}
