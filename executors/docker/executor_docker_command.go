package docker

import (
	"bytes"
	"fmt"

	"github.com/fsouza/go-dockerclient"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
)

type DockerCommandExecutor struct {
	DockerExecutor
}

func (s *DockerCommandExecutor) Start() error {
	s.Debugln("Starting Docker command...")

	// Create container
	err := s.createBuildContainer(s.BuildScript.GetCommandWithArguments())
	if err != nil {
		return err
	}

	// Wait for process to exit
	go func() {
		attachContainerOptions := docker.AttachToContainerOptions{
			Container:    s.buildContainer.ID,
			InputStream:  bytes.NewBufferString(s.BuildScript.Script),
			OutputStream: s.BuildLog,
			ErrorStream:  s.BuildLog,
			Logs:         true,
			Stream:       true,
			Stdin:        true,
			Stdout:       true,
			Stderr:       true,
			RawTerminal:  false,
		}

		s.Debugln("Attaching to container...")
		err := s.client.AttachToContainer(attachContainerOptions)
		if err != nil {
			s.BuildFinish <- err
			return
		}

		s.Debugln("Waiting for container...")
		exitCode, err := s.client.WaitContainer(s.buildContainer.ID)
		if err != nil {
			s.BuildFinish <- err
			return
		}

		if exitCode == 0 {
			s.BuildFinish <- nil
		} else {
			s.BuildFinish <- fmt.Errorf("exit code %d", exitCode)
		}
	}()
	return nil
}

func init() {
	options := executors.ExecutorOptions{
		DefaultBuildsDir: "/builds",
		SharedBuildsDir:  false,
		Shell: common.ShellScriptInfo{
			Shell: "bash",
			Type:  common.NormalShell,
		},
		ShowHostname:     true,
		SupportedOptions: []string{"image", "services"},
	}

	creator := func() common.Executor {
		return &DockerCommandExecutor{
			DockerExecutor: DockerExecutor{
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
		Creator: creator,
		FeaturesUpdater: featuresUpdater,
	})
}
