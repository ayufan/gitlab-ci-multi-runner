package src

import (
	"bytes"
	"errors"
	"os/exec"
)

type ShellExecutor struct {
	AbstractExecutor
	cmd *exec.Cmd
}

func (s *ShellExecutor) Prepare(config *RunnerConfig, build *Build) error {
	err := s.AbstractExecutor.Prepare(config, build)
	if err != nil {
		return err
	}

	s.println("Using Shell executor...")
	return nil
}

func (s *ShellExecutor) Start() error {
	s.println("Starting shell command...")

	shell_script := s.config.ShellScript
	if len(shell_script) == 0 {
		shell_script = "bash"
	}

	// Create execution command
	s.cmd = exec.Command(shell_script)
	if s.cmd == nil {
		return errors.New("Failed to generate execution command")
	}

	SetProcessGroup(s.cmd)

	// Fill process environment variables
	s.cmd.Env = append(s.build.GetEnv(), s.config.Environment...)
	s.cmd.Stdin = bytes.NewReader(s.BuildScript)
	s.cmd.Stdout = s.BuildLog
	s.cmd.Stderr = s.BuildLog

	// Start process
	err := s.cmd.Start()
	if err != nil {
		return errors.New("Failed to start process")
	}

	// Wait for process to exit
	go func() {
		s.buildFinish <- s.cmd.Wait()
	}()
	return nil
}

func (s *ShellExecutor) Cleanup() {
	KillProcessGroup(s.cmd)
	s.AbstractExecutor.Cleanup()
}
