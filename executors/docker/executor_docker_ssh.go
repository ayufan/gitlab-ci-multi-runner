package docker

import (
	"errors"

	"github.com/ayufan/gitlab-ci-multi-runner/common"
	"github.com/ayufan/gitlab-ci-multi-runner/executors"
	"github.com/ayufan/gitlab-ci-multi-runner/ssh"
)

type DockerSSHExecutor struct {
	DockerExecutor
	sshCommand ssh.Command
}

func (s *DockerSSHExecutor) Start() error {
	if s.Config.SSH == nil {
		return errors.New("Missing SSH configuration")
	}

	s.Debugln("Starting SSH command...")

	// Create container
	container, err := s.createContainer(s.image, []string{})
	if err != nil {
		return err
	}
	s.container = container

	containerData, err := s.client.InspectContainer(container.ID)
	if err != nil {
		return err
	}

	// Create SSH command
	s.sshCommand = ssh.Command{
		Config:      *s.Config.SSH,
		Environment: append(s.ShellScript.Environment, s.Config.Environment...),
		Command:     s.ShellScript.GetFullCommand(),
		Stdin:       s.ShellScript.Script,
		Stdout:      s.BuildLog,
		Stderr:      s.BuildLog,
	}
	s.sshCommand.Host = &containerData.NetworkSettings.IPAddress

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

func (s *DockerSSHExecutor) Cleanup() {
	s.sshCommand.Cleanup()
	s.DockerExecutor.Cleanup()
}

func init() {
	common.RegisterExecutor("docker-ssh", func() common.Executor {
		return &DockerSSHExecutor{
			DockerExecutor: DockerExecutor{
				AbstractExecutor: executors.AbstractExecutor{
					DefaultBuildsDir: "builds",
					SharedBuildsDir:  false,
					DefaultShell:     "bash",
					ShowHostname:     true,
				},
			},
		}
	})
}
