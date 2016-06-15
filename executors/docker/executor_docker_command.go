package docker

import (
	"bytes"
	"errors"

	"github.com/fsouza/go-dockerclient"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
)

type commandExecutor struct {
	executor
	predefinedContainer *docker.Container
	buildContainer      *docker.Container
}

func (s *commandExecutor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	err := s.executor.Prepare(globalConfig, config, build)
	if err != nil {
		return err
	}

	s.Debugln("Starting Docker command...")

	if len(s.BuildShell.DockerCommand) == 0 {
		return errors.New("Script is not compatible with Docker")
	}

	imageName, err := s.getImageName()
	if err != nil {
		return err
	}

	options, err := s.prepareBuildContainer()
	if err != nil {
		return err
	}

	buildImage, err := s.getPrebuiltImage()
	if err != nil {
		return err
	}

	// Start pre-build container which will git clone changes
	s.predefinedContainer, err = s.createContainer("predefined", buildImage.ID, []string{"gitlab-runner-build"}, *options)
	if err != nil {
		return err
	}

	// Start build container which will run actual build
	s.buildContainer, err = s.createContainer("build", imageName, s.BuildShell.DockerCommand, *options)
	if err != nil {
		return err
	}
	return nil
}

func (s *commandExecutor) Run(cmd common.ExecutorCommand) error {
	var container *docker.Container

	if cmd.Predefined {
		container = s.predefinedContainer
	} else {
		container = s.buildContainer
	}

	s.Debugln("Executing on", container.Name, "the", cmd.Script)

	return s.watchContainer(container, bytes.NewBufferString(cmd.Script), cmd.Abort)
}

func init() {
	options := executors.ExecutorOptions{
		DefaultBuildsDir: "/builds",
		DefaultCacheDir:  "/cache",
		SharedBuildsDir:  false,
		Shell: common.ShellScriptInfo{
			Shell:         "bash",
			Type:          common.NormalShell,
			RunnerCommand: "/usr/bin/gitlab-runner-helper",
		},
		ShowHostname:     true,
		SupportedOptions: []string{"image", "services"},
	}

	creator := func() common.Executor {
		return &commandExecutor{
			executor: executor{
				AbstractExecutor: executors.AbstractExecutor{
					ExecutorOptions: options,
				},
			},
		}
	}

	featuresUpdater := func(features *common.FeaturesInfo) {
		features.Variables = true
		features.Image = true
		features.Services = true
	}

	common.RegisterExecutor("docker", executors.DefaultExecutorProvider{
		Creator:         creator,
		FeaturesUpdater: featuresUpdater,
	})
}
