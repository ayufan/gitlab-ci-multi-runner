package docker

import (
	"bytes"
	"fmt"

	"github.com/fsouza/go-dockerclient"

	"github.com/ayufan/gitlab-ci-multi-runner/common"
	"github.com/ayufan/gitlab-ci-multi-runner/executors"
)

type DockerCommandExecutor struct {
	DockerExecutor
}

func (s *DockerCommandExecutor) Start() error {
	s.Debugln("Starting Docker command...")

	// Create container
	container, err := s.createContainer(s.image, s.ShellScript.GetCommandWithArguments())
	if err != nil {
		return err
	}
	s.container = container

	// Wait for process to exit
	go func() {
		attachContainerOptions := docker.AttachToContainerOptions{
			Container:    container.ID,
			InputStream:  bytes.NewBuffer(s.ShellScript.Script),
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
		exitCode, err := s.client.WaitContainer(container.ID)
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
	common.RegisterExecutor("docker", func() common.Executor {
		return &DockerCommandExecutor{
			DockerExecutor: DockerExecutor{
				AbstractExecutor: executors.AbstractExecutor{
					DefaultBuildsDir: "/builds",
					SharedBuildsDir:  false,
					DefaultShell:     "bash",
					ShowHostname:     true,
				},
			},
		}
	})
}
