package docker

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/fsouza/go-dockerclient"

	"github.com/ayufan/gitlab-ci-multi-runner/common"
	"github.com/ayufan/gitlab-ci-multi-runner/executors"
)

type DockerCommandExecutor struct {
	DockerExecutor
}

func (s *DockerCommandExecutor) Start() error {
	s.Println("Starting Docker command...")

	// Create container
	container, err := s.createContainer(s.image, []string{"bash"})
	if err != nil {
		return err
	}
	s.container = container

	// Wait for process to exit
	go func() {
		attach_container_opts := docker.AttachToContainerOptions{
			Container:    container.ID,
			InputStream:  bytes.NewBuffer(s.BuildScript),
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
		err := s.client.AttachToContainer(attach_container_opts)
		if err != nil {
			s.BuildFinish <- err
			return
		}

		s.Debugln("Waiting for container...")
		exit_code, err := s.client.WaitContainer(container.ID)
		if err != nil {
			s.BuildFinish <- err
			return
		}

		if exit_code == 0 {
			s.BuildFinish <- nil
		} else {
			s.BuildFinish <- errors.New(fmt.Sprintf("exit code %d", exit_code))
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
					ShowHostname:     true,
				},
			},
		}
	})
}
