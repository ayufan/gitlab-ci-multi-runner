package src

import (
	"bytes"
	"errors"
	"os/exec"
)

type ShellExecutor struct {
	BaseExecutor
}

func (s *ShellExecutor) Start() error {
	shell_script := s.config.ShellScript
	if len(shell_script) == 0 {
		shell_script = "setsid"
	}

	// Create execution command
	cmd := exec.Command(shell_script, "bash", "--login")
	if cmd == nil {
		return errors.New("Failed to generate execution command")
	}

	cmd.Env = append(s.build.GetEnv(), s.config.Environment...)
	cmd.Stdin = bytes.NewReader(s.script_data)
	cmd.Stdout = s.build_log
	cmd.Stderr = s.build_log

	// Start process
	err := cmd.Start()
	if err != nil {
		return errors.New("Failed to start process")
	}

	// Wait for process to exit
	go func() {
		s.buildFinish <- cmd.Wait()
	}()

	s.buildAbortFunc = func(e *BaseExecutor) {
		cmd.Process.Kill()
	}
	return nil
}
