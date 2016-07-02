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
	executorOptions = executors.ExecutorOptions{
		SharedBuildsDir: false,
		Shell: common.ShellScriptInfo{
			Shell:         "bash",
			Type:          common.NormalShell,
			RunnerCommand: "/usr/bin/gitlab-runner-helper",
		},
		ShowHostname:     true,
		SupportedOptions: []string{"image", "services", "artifacts", "cache"},
	}
)

type kubernetesOptions struct {
	Image    string   `json:"image"`
	Services []string `json:"services"`
}

type executor struct {
	executors.AbstractExecutor

	kubeClient   *client.Client
	prepod       *api.Pod
	pod          *api.Pod
	options      *kubernetesOptions
	extraOptions Options

	buildLimits   api.ResourceList
	serviceLimits api.ResourceList
}

func (s *executor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	err := s.AbstractExecutor.Prepare(globalConfig, config, build)
	if err != nil {
		return err
	}

	if s.BuildScript.PassFile {
		return fmt.Errorf("Kubernetes doesn't support shells that require script file")
	}

	s.extraOptions = DefaultOptions{s.Build.Variables}

	if !s.Config.Kubernetes.AllowPrivileged && s.extraOptions.Privileged() {
		return fmt.Errorf("Runner does not allow privileged containers")
	}

	err = build.Options.Decode(&s.options)
	if err != nil {
		return err
	}

	s.kubeClient, err = getKubeClient(config.Kubernetes)
	if err != nil {
		return fmt.Errorf("Error connecting to Kubernetes: %s", err.Error())
	}

	if s.serviceLimits, err = limits(s.Config.Kubernetes.ServiceCPUs, s.Config.Kubernetes.ServiceMemory); err != nil {
		return err
	}

	if s.buildLimits, err = limits(s.Config.Kubernetes.CPUs, s.Config.Kubernetes.Memory); err != nil {
		return err
	}

	s.Println("Using Kubernetes executor with image", s.options.Image, "...")

	return nil
}

func (s *executor) Run(cmd common.ExecutorCommand) error {
	var err error
	s.Debugln("Starting Kubernetes command...")

	if s.pod == nil {
		services := make([]api.Container, len(s.options.Services))
		for i, image := range s.options.Services {
			services[i] = s.buildContainer(fmt.Sprintf("svc-%d", i), image, s.serviceLimits)
		}

		s.pod, err = s.kubeClient.Pods(s.Config.Kubernetes.Namespace).Create(&api.Pod{
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
					s.buildContainer("build", s.options.Image, s.buildLimits, s.BuildScript.DockerCommand...),
					s.buildContainer("pre", "munnerz/gitlab-runner-helper", s.serviceLimits, s.BuildScript.DockerCommand...),
				}, services...),
			},
		})

		if err != nil {
			return err
		}
	}

	var containerName string
	switch {
	case cmd.Predefined:
		containerName = "pre"
	default:
		containerName = "build"
	}

	select {
	case err := <-s.runInContainer(containerName, cmd.Script):
		return err
	case _ = <-cmd.Abort:
		return fmt.Errorf("build aborted")
	}
}

func (s *executor) Cleanup() {
	if s.pod != nil {
		err := s.kubeClient.Pods(s.pod.Namespace).Delete(s.pod.Name, nil)

		if err != nil {
			s.Errorln(fmt.Sprintf("Error cleaning up pod: %s", err.Error()))
		}
	}
	s.AbstractExecutor.Cleanup()
}

func (s *executor) buildContainer(name, image string, limits api.ResourceList, command ...string) api.Container {
	path := strings.Split(s.Shell.Build.BuildDir, "/")
	path = path[:len(path)-1]

	privileged := s.extraOptions.Privileged()

	return api.Container{
		Name:    name,
		Image:   image,
		Command: command,
		Env:     buildVariables(s.Build.GetAllVariables().PublicOrInternal()),
		Resources: api.ResourceRequirements{
			Limits: limits,
		},
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

func (s *executor) runInContainer(name, command string) <-chan error {
	errc := make(chan error, 1)
	go func() {
		defer close(errc)

		status, err := waitForPodRunning(s.kubeClient, s.pod, s.BuildLog)

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

		exec := ExecOptions{
			PodName:       s.pod.Name,
			Namespace:     s.pod.Namespace,
			ContainerName: name,
			Command:       s.BuildScript.DockerCommand,
			In:            strings.NewReader(command),
			Out:           s.BuildLog,
			Err:           s.BuildLog,
			Stdin:         true,
			Config:        config,
			Client:        s.kubeClient,
			Executor:      &DefaultRemoteExecutor{},
		}

		errc <- exec.Run()
	}()

	return errc
}

func createFn() common.Executor {
	return &executor{
		AbstractExecutor: executors.AbstractExecutor{
			ExecutorOptions: executorOptions,
		},
	}
}

func featuresFn(features *common.FeaturesInfo) {
	features.Variables = true
	features.Image = true
	features.Services = true
	features.Artifacts = true
	features.Cache = true
}

func init() {
	common.RegisterExecutor("kubernetes", executors.DefaultExecutorProvider{
		Creator:         createFn,
		FeaturesUpdater: featuresFn,
	})
}
