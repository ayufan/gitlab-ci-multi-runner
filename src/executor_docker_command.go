package src

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/fsouza/go-dockerclient"
)

type DockerCommandExecutor struct {
	DockerExecutor
}

func (s *DockerCommandExecutor) Start() error {
	s.println("Starting Docker command...")

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
			InputStream:  bytes.NewBuffer(s.script_data),
			OutputStream: s.BuildLog,
			ErrorStream:  s.BuildLog,
			Logs:         true,
			Stream:       true,
			Stdin:        true,
			Stdout:       true,
			Stderr:       true,
			RawTerminal:  false,
		}

		s.debugln("Attaching to container...")
		err := s.client.AttachToContainer(attach_container_opts)
		if err != nil {
			s.buildFinish <- err
			return
		}

		s.debugln("Waiting for container...")
		exit_code, err := s.client.WaitContainer(container.ID)
		if err != nil {
			s.buildFinish <- err
			return
		}

		if exit_code == 0 {
			s.buildFinish <- nil
		} else {
			s.buildFinish <- errors.New(fmt.Sprintf("exit code %d", exit_code))
		}
	}()
	return nil
}
