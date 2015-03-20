package docker

import (
	"errors"

	"github.com/ayufan/gitlab-ci-multi-runner/common"
	"github.com/ayufan/gitlab-ci-multi-runner/executors"
	"github.com/ayufan/gitlab-ci-multi-runner/ssh"
)

type DockerSshExecutor struct {
	DockerExecutor
	sshCommand ssh.SshCommand
}

func (s *DockerSshExecutor) Start() error {
	if s.Config.Ssh == nil {
		return errors.New("Missing SSH configuration")
	}

	s.Debugln("Starting SSH command...")

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
	s.sshCommand = ssh.SshCommand{
		SshConfig:   *s.Config.Ssh,
		Environment: append(s.BuildEnv, s.Config.Environment...),
		Command:     "bash",
		Stdin:       s.BuildScript,
		Stdout:      s.BuildLog,
		Stderr:      s.BuildLog,
	}
	s.sshCommand.Host = container_data.NetworkSettings.IPAddress

	s.Debugln("Connecting to SSH server...")
	err = s.sshCommand.Connect()
	if err != nil {
		return err
	}

	// Wait for process to exit
	go func() {
		s.Debugln("Will run SSH command...")
		err := s.sshCommand.Run()
		s.Debugln("SSH command finished with", err)
		s.BuildFinish <- err
	}()
	return nil
}

func (s *DockerSshExecutor) Cleanup() {
	s.sshCommand.Cleanup()
	s.DockerExecutor.Cleanup()
}

func init() {
	common.RegisterExecutor("docker-ssh", func() common.Executor {
		return &DockerSshExecutor{
			DockerExecutor: DockerExecutor{
				AbstractExecutor: executors.AbstractExecutor{
					DefaultBuildsDir: "builds",
					ShowHostname:     true,
				},
			},
		}
	})
}
