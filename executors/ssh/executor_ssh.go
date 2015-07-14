package ssh

import (
	"errors"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/ssh"
)

type SSHExecutor struct {
	executors.AbstractExecutor
	sshCommand ssh.Command
}

func (s *SSHExecutor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	err := s.AbstractExecutor.Prepare(globalConfig, config, build)
	if err != nil {
		return err
	}

	if s.ShellScript.PassFile {
		return errors.New("Parallels doesn't support shells that require script file")
	}
	return nil
}

func (s *SSHExecutor) Start() error {
	if s.Config.SSH == nil {
		return errors.New("Missing SSH configuration")
	}

	s.Debugln("Starting SSH command...")

	// Create SSH command
	s.sshCommand = ssh.Command{
		Config:      *s.Config.SSH,
		Environment: append(s.ShellScript.Environment, s.Config.Environment...),
		Command:     s.ShellScript.GetFullCommand(),
		Stdin:       s.ShellScript.GetScriptBytes(),
		Stdout:      s.BuildLog,
		Stderr:      s.BuildLog,
	}

	s.Debugln("Connecting to SSH server...")
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

func (s *SSHExecutor) Cleanup() {
	s.sshCommand.Cleanup()
	s.AbstractExecutor.Cleanup()
}

func init() {
	options := executors.ExecutorOptions{
		DefaultBuildsDir: "builds",
		SharedBuildsDir:  true,
		Shell: common.ShellScriptInfo{
			Shell:       "bash",
			Type:        common.LoginShell,
		},
		ShowHostname:     true,
	}

	create := func() common.Executor {
		return &SSHExecutor{
			AbstractExecutor: executors.AbstractExecutor{
				ExecutorOptions: options,
			},
		}
	}

	common.RegisterExecutor("ssh", common.ExecutorFactory{
		Create: create,
	})
}
