package src

import (
	"errors"
)

type SshExecutor struct {
	AbstractExecutor
	sshCommand SshCommand
}

func (s *SshExecutor) Start() error {
	if s.config.Ssh == nil {
		return errors.New("Missing SSH configuration")
	}

	s.println("Starting SSH command...")

	// Create SSH command
	s.sshCommand = SshCommand{
		SshConfig:   *s.config.Ssh,
		Environment: append(s.build.GetEnv(), s.config.Environment...),
		Command:     "bash",
		Stdin:       s.BuildScript,
		Stdout:      s.BuildLog,
		Stderr:      s.BuildLog,
	}

	s.println("Connecting to SSH server...")
	err := s.sshCommand.Connect()
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

func (s *SshExecutor) Cleanup() {
	s.sshCommand.Cleanup()
	s.AbstractExecutor.Cleanup()
}
