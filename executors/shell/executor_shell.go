package shell

import (
	"bytes"
	"errors"
	"os"
	"os/exec"

	"github.com/ayufan/gitlab-ci-multi-runner/common"
	"github.com/ayufan/gitlab-ci-multi-runner/executors"
	"github.com/ayufan/gitlab-ci-multi-runner/helpers"
)

type ShellExecutor struct {
	executors.AbstractExecutor
	cmd *exec.Cmd
}

func (s *ShellExecutor) Prepare(config *common.RunnerConfig, build *common.Build) error {
	err := s.AbstractExecutor.Prepare(config, build)
	if err != nil {
		return err
	}

	s.Println("Using Shell executor...")
	return nil
}

func (s *ShellExecutor) Start() error {
	s.Debugln("Starting shell command...")

	shell_script := s.Config.ShellScript
	if len(shell_script) == 0 {
		shell_script = "bash"
	}

	// Create execution command
	s.cmd = exec.Command(shell_script)
	if s.cmd == nil {
		return errors.New("Failed to generate execution command")
	}

	helpers.SetProcessGroup(s.cmd)

	// Inherit environment from current process
	if !s.Config.CleanEnvironment {
		s.cmd.Env = os.Environ()
	}

	// Fill process environment variables
	s.cmd.Env = append(s.cmd.Env, s.Build.GetEnv()...)
	s.cmd.Env = append(s.cmd.Env, s.Config.Environment...)
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
		s.BuildFinish <- s.cmd.Wait()
	}()
	return nil
}

func (s *ShellExecutor) Cleanup() {
	helpers.KillProcessGroup(s.cmd)
	s.AbstractExecutor.Cleanup()
}

func init() {
	common.RegisterExecutor("shell", func() common.Executor {
		return &ShellExecutor{
			AbstractExecutor: executors.AbstractExecutor{
				DefaultBuildsDir: "tmp/builds",
				ShowHostname:     false,
			},
		}
	})
}
