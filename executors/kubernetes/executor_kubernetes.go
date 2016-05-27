package kubernetes

import (
	"errors"
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
	// TODO: Support a whitelist of images that can be privileged?
	Privileged bool `json:"pivileged"`
}

type executor struct {
	executors.AbstractExecutor

	pod             *api.Pod
	scriptConfigMap *api.ConfigMap
	options         *kubernetesOptions
}

func (s *executor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	err := s.AbstractExecutor.Prepare(globalConfig, config, build)
	if err != nil {
		return err
	}

	fmt.Printf("Got runner config: %s\n", config)
	fmt.Printf("Got build: %s\n", build)
	var allowPrivileged bool
	if config.Kubernetes != nil {
		allowPrivileged = config.Kubernetes.AllowPrivileged
	}

	if kubeClient == nil {
		kubeClient, err = getKubeClient(config.Kubernetes)
		if err != nil {
			return err
		}
	}

	if s.BuildScript.PassFile {
		return errors.New("Kubernetes doesn't support shells that require script file")
	}

	err = build.Options.Decode(&s.options)
	if err != nil {
		return err
	}

	if !allowPrivileged && s.options.Privileged {
		return errors.New("Attempting to run privileged container but runner config does not allow it")
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

func (s *executor) Run(cmd common.ExecutorCommand) error {
	s.Debugln("Starting Kubernetes command...")

	if cmd.Predefined {
		// We don't currently use the pre/post containers
		return nil
	}

	if s.pod == nil {
		var err error
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
							GitRepo: &api.GitRepoVolumeSource{
								Repository: s.Shell.Build.RepoURL,
							},
						},
					},
				},
				RestartPolicy: api.RestartPolicyNever,
				Containers: []api.Container{
					api.Container{
						Name: "build",
						// TODO: clean image name
						Command: []string{"bash"},
						Image:   s.options.Image,
						VolumeMounts: []api.VolumeMount{
							api.VolumeMount{
								Name:      "repo",
								MountPath: s.Shell.Build.BuildDir,
							},
						},
						SecurityContext: &api.SecurityContext{
							Privileged: &s.options.Privileged,
						},
						Stdin: true,
					},
				},
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

			exec := ExecOptions{
				PodName:       s.pod.Name,
				Namespace:     s.pod.Namespace,
				ContainerName: "build",
				Command:       []string{"bash"},
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
			Shell: "bash",
			Type:  common.NormalShell,
		},
		ShowHostname:     true,
		SupportedOptions: []string{"image", "services"},
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
	}

	common.RegisterExecutor("kubernetes", executors.DefaultExecutorProvider{
		Creator:         creator,
		FeaturesUpdater: featuresUpdater,
	})
}
