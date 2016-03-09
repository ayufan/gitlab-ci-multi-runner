package ssh

import (
	"errors"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/ssh"
)

type executor struct {
	executors.AbstractExecutor
	sshCommand ssh.Command
}

func (s *executor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	err := s.AbstractExecutor.Prepare(globalConfig, config, build)
	if err != nil {
		return err
	}

	s.Println("Using SSH executor...")
	if s.BuildScript.PassFile {
		return errors.New("SSH doesn't support shells that require script file")
	}
	return nil
}

func (s *executor) Start() error {
	if s.Config.SSH == nil {
		return errors.New("Missing SSH configuration")
	}

	s.Debugln("Starting SSH command...")

	// Create SSH command
	s.sshCommand = ssh.Command{
		Config:      *s.Config.SSH,
		Environment: s.BuildScript.Environment,
		Command:     s.BuildScript.GetCommandWithArguments(),
		Stdin:       s.BuildScript.GetScriptBytes(),
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

func (s *executor) Cleanup() {
	s.sshCommand.Cleanup()
	s.AbstractExecutor.Cleanup()
}

func init() {
	options := executors.ExecutorOptions{
		DefaultBuildsDir: "builds",
		SharedBuildsDir:  true,
		Shell: common.ShellScriptInfo{
			Shell: "bash",
			Type:  common.LoginShell,
		},
		ShowHostname: true,
	}

	creator := func() common.Executor {
		return &executor{
			AbstractExecutor: executors.AbstractExecutor{
				ExecutorOptions: options,
			},
		}
	}

	featuresUpdater := func(features *common.FeaturesInfo) {
		features.Variables = true
	}

	common.RegisterExecutor("ssh", executors.DefaultExecutorProvider{
		Creator:         creator,
		FeaturesUpdater: featuresUpdater,
	})
}
