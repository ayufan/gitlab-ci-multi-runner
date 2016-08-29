package kubernetes

import (
	"fmt"
	"strings"

	"golang.org/x/net/context"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
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

	kubeClient *client.Client
	prepod     *api.Pod
	pod        *api.Pod
	options    *kubernetesOptions

	buildLimits   api.ResourceList
	serviceLimits api.ResourceList
}

func (s *executor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	err := s.AbstractExecutor.Prepare(globalConfig, config, build)
	if err != nil {
		return err
	}

	if s.BuildShell.PassFile {
		return fmt.Errorf("kubernetes doesn't support shells that require script file")
	}

	err = build.Options.Decode(&s.options)
	if err != nil {
		return err
	}

	s.kubeClient, err = getKubeClient(config.Kubernetes)
	if err != nil {
		return fmt.Errorf("error connecting to Kubernetes: %s", err.Error())
	}

	if s.serviceLimits, err = limits(s.Config.Kubernetes.ServiceCPUs, s.Config.Kubernetes.ServiceMemory); err != nil {
		return err
	}

	if s.buildLimits, err = limits(s.Config.Kubernetes.CPUs, s.Config.Kubernetes.Memory); err != nil {
		return err
	}

	if err = s.checkDefaults(); err != nil {
		return err
	}

	s.Println("Using Kubernetes executor with image", s.options.Image, "...")

	return nil
}

func (s *executor) Run(cmd common.ExecutorCommand) error {
	s.Debugln("Starting Kubernetes command...")

	if s.pod == nil {
		err := s.setupBuildPod()

		if err != nil {
			return err
		}
	}

	containerName := "build"

	ctx, cancel := context.WithCancel(context.Background())
	select {
	case err := <-s.runInContainer(ctx, containerName, cmd.Script):
		if err != nil && strings.Contains(err.Error(), "executing in Docker Container") {
			return &common.BuildError{Inner: err}
		}
		return err
	case <-cmd.Abort:
		cancel()
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
	closeKubeClient(s.kubeClient)
	s.AbstractExecutor.Cleanup()
}

func (s *executor) buildContainer(name, image string, limits api.ResourceList, command ...string) api.Container {
	path := strings.Split(s.Build.BuildDir, "/")
	path = path[:len(path)-1]

	privileged := false
	if s.Config.Kubernetes != nil {
		privileged = s.Config.Kubernetes.Privileged
	}

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

func (s *executor) setupBuildPod() error {
	services := make([]api.Container, len(s.options.Services))
	for i, image := range s.options.Services {
		resolvedImage := s.Build.GetAllVariables().ExpandValue(image)
		services[i] = s.buildContainer(fmt.Sprintf("svc-%d", i), resolvedImage, s.serviceLimits)
	}

	buildImage := s.Build.GetAllVariables().ExpandValue(s.options.Image)
	pod, err := s.kubeClient.Pods(s.Config.Kubernetes.Namespace).Create(&api.Pod{
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
				s.buildContainer("build", buildImage, s.buildLimits, s.BuildShell.DockerCommand...),
			}, services...),
		},
	})

	if err != nil {
		return err
	}

	s.pod = pod

	return nil
}

func (s *executor) runInContainer(ctx context.Context, name, command string) <-chan error {
	errc := make(chan error, 1)
	go func() {
		defer close(errc)

		status, err := waitForPodRunning(ctx, s.kubeClient, s.pod, s.BuildTrace)

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
			Command:       s.BuildShell.DockerCommand,
			In:            strings.NewReader(command),
			Out:           s.BuildTrace,
			Err:           s.BuildTrace,
			Stdin:         true,
			Config:        config,
			Client:        s.kubeClient,
			Executor:      &DefaultRemoteExecutor{},
		}

		errc <- exec.Run()
	}()

	return errc
}

func (s *executor) checkDefaults() error {
	if s.options.Image == "" {
		if s.Config.Kubernetes.Image == "" {
			return fmt.Errorf("no image specified and no default set in config")
		}

		s.options.Image = s.Config.Kubernetes.Image
	}

	if s.Config.Kubernetes.Namespace == "" {
		s.Config.Kubernetes.Namespace = "default"
	}

	return nil
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
