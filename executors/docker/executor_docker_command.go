package docker

import (
	"bytes"
	"errors"
	"github.com/fsouza/go-dockerclient"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
	"strconv"
)

type commandExecutor struct {
	executor
	defaultImage     string
	defaultContainer *docker.CreateContainerOptions
	containerIdx     int
}

func (s *commandExecutor) Prepare(build *common.Build, data common.ExecutorData) (err error) {
	err = s.executor.Prepare(build, data)
	if err != nil {
		return
	}

	s.defaultImage, err = s.getImageName()
	if err != nil {
		return
	}

	s.defaultContainer, err = s.prepareBuildContainer()
	if err != nil {
		return
	}

	return
}

func (s *commandExecutor) getContainerImage(runImage string) (string, error) {
	switch runImage {
	case common.ImagePreBuild, common.ImagePostBuild:
		buildImage, err := s.getPrebuiltImage("build")
		if err != nil {
			return "", nil
		}
		return buildImage.ID, nil

	case common.ImageDefault:
		return s.defaultImage, nil

	default:
		return runImage, nil
	}

	return "", errors.New("undefined run type: " + runImage)
}

func (s *commandExecutor) Run(run common.ExecutorRun) (err error) {
	containerImage, err := s.getContainerImage(run.Image)
	if err != nil {
		return
	}

	containerCommand := append([]string{run.Command}, run.Arguments...)

	s.containerIdx++
	containerType := "step-" + strconv.Itoa(s.containerIdx)

	// Start pre-build container which will git clone changes
	container, err := s.createContainer(containerType, containerImage, containerCommand, *s.defaultContainer)
	if err != nil {
		return err
	}
	defer s.removeContainer(container.ID)

	resultCh := make(chan error, 1)
	go func() {
		resultCh <- s.watchContainer(container, bytes.NewBufferString(run.Script), run.Trace)
	}()

	// Wait for process to finish
	select {
	case err = <-resultCh:
		return
	case err = <-run.Abort:
		return
	}
	return
}

func init() {
	options := executors.ExecutorOptions{
		DefaultBuildsDir: "/builds",
		DefaultCacheDir:  "/cache",
		SharedBuildsDir:  false,
		Shell:            "bash",
		Type:             common.NormalShell,
		RunnerCommand:    "/usr/bin/gitlab-runner-helper",
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
