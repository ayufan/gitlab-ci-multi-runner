package docker

import (
	"errors"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/ssh"
)

type sshExecutor struct {
	executor
	sshCommand ssh.Command
}

func (s *sshExecutor) Start() error {
	if s.Config.SSH == nil {
		return errors.New("Missing SSH configuration")
	}

	s.Debugln("Starting SSH command...")

	imageName, err := s.getImageName()
	if err != nil {
		return err
	}

	options, err := s.prepareBuildContainer()
	if err != nil {
		return err
	}

	// Start build container which will run actual build
	container, err := s.createContainer("build", imageName, []string{}, *options)
	if err != nil {
		return err
	}

	s.Debugln("Starting container", container.ID, "...")
	err = s.client.StartContainer(container.ID, nil)
	if err != nil {
		return err
	}

	containerData, err := s.client.InspectContainer(container.ID)
	if err != nil {
		return err
	}

	// Create SSH command
	s.sshCommand = ssh.Command{
		Config:      *s.Config.SSH,
		Environment: s.BuildScript.Environment,
		Command:     s.BuildScript.GetCommandWithArguments(),
		Stdin:       s.BuildScript.GetScriptBytes(),
		Stdout:      s.BuildLog,
		Stderr:      s.BuildLog,
	}
	s.sshCommand.Host = containerData.NetworkSettings.IPAddress

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

func (s *sshExecutor) Cleanup() {
	s.sshCommand.Cleanup()
	s.executor.Cleanup()
}

func init() {
	options := executors.ExecutorOptions{
		DefaultBuildsDir: "builds",
		SharedBuildsDir:  false,
		Shell: common.ShellScriptInfo{
			Shell: "bash",
			Type:  common.LoginShell,
		},
		ShowHostname:     true,
		SupportedOptions: []string{"image", "services"},
	}

	creator := func() common.Executor {
		return &sshExecutor{
			executor: executor{
				AbstractExecutor: executors.AbstractExecutor{
					ExecutorOptions: options,
				},
			},
		}
	}

	featuresUpdater := func(features *common.FeaturesInfo) {
		features.Variables = true
		features.Image = true
		features.Services = true
	}

	common.RegisterExecutor("docker-ssh", executors.DefaultExecutorProvider{
		Creator:         creator,
		FeaturesUpdater: featuresUpdater,
	})
}
