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
}

func (s *commandExecutor) watchContainers(preContainer, buildContainer, postContainer *docker.Container) {
	s.Println()

	err := s.watchContainer(preContainer, bytes.NewBufferString(s.BuildScript.PreScript))
	if err != nil {
		s.BuildFinish <- err
		return
	}

	s.Println()

	err = s.watchContainer(buildContainer, bytes.NewBufferString(s.BuildScript.BuildScript))
	if err != nil {
		s.BuildFinish <- err
		return
	}

	s.Println()

	err = s.watchContainer(postContainer, bytes.NewBufferString(s.BuildScript.PostScript))
	if err != nil {
		s.BuildFinish <- err
		return
	}

	s.BuildFinish <- nil
}

func (s *commandExecutor) Start() error {
	s.Debugln("Starting Docker command...")

	imageName, err := s.getImageName()
	if err != nil {
		return err
	}

	options, err := s.prepareBuildContainer()
	if err != nil {
		return err
	}

	buildImage, err := s.getPrebuiltImage("build")
	if err != nil {
		return err
	}

	// Start pre-build container which will git clone changes
	preContainer, err := s.createContainer("pre", buildImage.ID, nil, *options)
	if err != nil {
		return err
	}

	// Start post-build container which will upload artifacts
	postContainer, err := s.createContainer("post", buildImage.ID, nil, *options)
	if err != nil {
		return err
	}

	if len(s.BuildScript.DockerCommand) == 0 {
		return errors.New("Script is not compatible with Docker")
	}

	// Start build container which will run actual build
	buildContainer, err := s.createContainer("build", imageName, s.BuildScript.DockerCommand, *options)
	if err != nil {
		return err
	}

	// Wait for process to exit
	go s.watchContainers(preContainer, buildContainer, postContainer)
	return nil
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
