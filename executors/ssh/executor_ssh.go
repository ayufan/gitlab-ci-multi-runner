package ssh

import (
	"errors"

	"github.com/ayufan/gitlab-ci-multi-runner/executors"
	"github.com/ayufan/gitlab-ci-multi-runner/ssh"
)

type SshExecutor struct {
	executors.AbstractExecutor
	sshCommand ssh.SshCommand
}

func (s *SshExecutor) Start() error {
	if s.Config.Ssh == nil {
		return errors.New("Missing SSH configuration")
	}

	s.Println("Starting SSH command...")

	// Create SSH command
	s.sshCommand = ssh.SshCommand{
		SshConfig:   *s.Config.Ssh,
		Environment: append(s.Build.GetEnv(), s.Config.Environment...),
		Command:     "bash",
		Stdin:       s.BuildScript,
		Stdout:      s.BuildLog,
		Stderr:      s.BuildLog,
	}

	s.Println("Connecting to SSH server...")
	err := s.sshCommand.Connect()
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

func (s *SshExecutor) Cleanup() {
	s.sshCommand.Cleanup()
	s.AbstractExecutor.Cleanup()
}
