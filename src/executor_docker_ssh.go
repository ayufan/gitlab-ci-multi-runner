package src

import (
	"errors"
)

type DockerSshExecutor struct {
	DockerExecutor
	sshCommand SshCommand
}

func (s *DockerSshExecutor) Start() error {
	if s.config.Ssh == nil {
		return errors.New("Missing SSH configuration")
	}

	s.debugln("Starting SSH command...")

	// Create container
	container, err := s.createContainer(s.image, []string{})
	if err != nil {
		return err
	}
	s.container = container

	container_data, err := s.client.InspectContainer(container.ID)
	if err != nil {
		return err
	}

	// Create SSH command
	s.sshCommand = SshCommand{
		SshConfig:   *s.config.Ssh,
		Environment: append(s.build.GetEnv(), s.config.Environment...),
		Command:     "bash",
		Stdin:       s.BuildScript,
		Stdout:      s.BuildLog,
		Stderr:      s.BuildLog,
	}
	s.sshCommand.Host = container_data.NetworkSettings.IPAddress

	s.println("Connecting to SSH server...")
	err = s.sshCommand.Connect()
	if err != nil {
		return err
	}

	// Wait for process to exit
	go func() {
		s.debugln("Will run SSH command...")
		err := s.sshCommand.Run()
		s.debugln("SSH command finished with", err)
		s.buildFinish <- err
	}()
	return nil
}

func (s *DockerSshExecutor) Cleanup() {
	s.sshCommand.Cleanup()
	s.DockerExecutor.Cleanup()
}
