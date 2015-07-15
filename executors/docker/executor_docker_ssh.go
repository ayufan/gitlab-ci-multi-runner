package docker

import (
	"errors"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/ssh"
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
		Environment: s.ShellScript.Environment,
		Command:     s.ShellScript.GetFullCommand(),
		Stdin:       s.ShellScript.GetScriptBytes(),
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

	create := func() common.Executor {
		return &DockerSSHExecutor{
			DockerExecutor: DockerExecutor{
				AbstractExecutor: executors.AbstractExecutor{
					ExecutorOptions: options,
				},
			},
		}
	}

	common.RegisterExecutor("docker-ssh", common.ExecutorFactory{
		Create: create,
		Features: common.FeaturesInfo{
			Variables: true,
			Image:     true,
			Services:  true,
		},
	})
}
