package virtualbox

import (
	"errors"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/ssh"
	vbox "gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/virtualbox"
	"time"
)

type executor struct {
	executors.AbstractExecutor
	vmName          string
	sshCommand      ssh.Client
	sshPort         string
	provisioned     bool
	machineVerified bool
}

func (s *executor) verifyMachine(vmName string, sshPort string) error {
	if s.machineVerified {
		return nil
	}

	// Create SSH command
	sshCommand := ssh.Client{
		Config:         *s.Config.SSH,
		Stdout:         s.BuildTrace,
		Stderr:         s.BuildTrace,
		ConnectRetries: 30,
	}
	sshCommand.Port = sshPort
	sshCommand.Host = "localhost"

	s.Debugln("Connecting to SSH...")
	err := sshCommand.Connect()
	if err != nil {
		return err
	}
	defer sshCommand.Cleanup()
	err = sshCommand.Run(ssh.Command{Command: []string{"exit"}})
	if err != nil {
		return err
	}
	s.machineVerified = true
	return nil
}

func (s *executor) restoreFromSnapshot() error {
	s.Debugln("Reverting VM to current snapshot...")
	err := vbox.RevertToSnapshot(s.vmName)
	if err != nil {
		return err
	}

	return nil
}

func (s *executor) determineBaseSnapshot(baseImage string) string {
	var err error
	baseSnapshot := s.Config.VirtualBox.BaseSnapshot
	if baseSnapshot == "" {
		baseSnapshot, err = vbox.GetCurrentSnapshot(baseImage)
		if err != nil {
			if s.Config.VirtualBox.DisableSnapshots {
				s.Debugln("No snapshots found for base VM", baseImage)
				return ""
			}

			baseSnapshot = "Base State"
		}
	}

	if baseSnapshot != "" && !vbox.HasSnapshot(baseImage, baseSnapshot) {
		if s.Config.VirtualBox.DisableSnapshots {
			s.Warningln("Snapshot", baseSnapshot, "not found in base VM", baseImage)
			return ""
		}

		s.Debugln("Creating snapshot", baseSnapshot, "from current base VM", baseImage, "state...")
		err = vbox.CreateSnapshot(baseImage, baseSnapshot)
		if err != nil {
			s.Warningln("Failed to create snapshot", baseSnapshot, "from base VM", baseImage)
			return ""
		}
	}

	return baseSnapshot
}

// virtualbox doesn't support templates
func (s *executor) createVM(vmName string) (err error) {
	baseImage := s.Config.VirtualBox.BaseName
	if baseImage == "" {
		return errors.New("Missing Image setting from VirtualBox configuration")
	}

	_, err = vbox.Status(vmName)
	if err != nil {
		vbox.Unregister(vmName)
	}

	if !vbox.Exist(vmName) {
		baseSnapshot := s.determineBaseSnapshot(baseImage)
		if baseSnapshot == "" {
			s.Debugln("Creating testing VM from VM", baseImage, "...")
		} else {
			s.Debugln("Creating testing VM from VM", baseImage, "snapshot", baseSnapshot, "...")
		}

		err = vbox.CreateOsVM(baseImage, vmName, baseSnapshot)
		if err != nil {
			return err
		}
	}

	s.Debugln("Identify SSH Port...")
	s.sshPort, err = vbox.FindSSHPort(s.vmName)
	if err != nil {
		s.Debugln("Creating localhost ssh forwarding...")
		vmSSHPort := s.Config.SSH.Port
		if vmSSHPort == "" {
			vmSSHPort = "22"
		}
		s.sshPort, err = vbox.ConfigureSSH(vmName, vmSSHPort)
		if err != nil {
			return err
		}
	}
	s.Debugln("Using local", s.sshPort, "SSH port to connect to VM...")

	s.Debugln("Bootstraping VM...")
	err = vbox.Start(s.vmName)
	if err != nil {
		return err
	}

	s.Debugln("Waiting for VM to become responsive...")
	time.Sleep(10)
	err = s.verifyMachine(s.vmName, s.sshPort)
	if err != nil {
		return err
	}

	return nil
}

func (s *executor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	err := s.AbstractExecutor.Prepare(globalConfig, config, build)
	if err != nil {
		return err
	}

	if s.BuildShell.PassFile {
		return errors.New("virtualbox doesn't support shells that require script file")
	}

	if s.Config.SSH == nil {
		return errors.New("Missing SSH config")
	}

	if s.Config.VirtualBox == nil {
		return errors.New("Missing VirtualBox configuration")
	}

	if s.Config.VirtualBox.BaseName == "" {
		return errors.New("Missing BaseName setting from VirtualBox configuration")
	}

	version, err := vbox.Version()
	if err != nil {
		return err
	}

	s.Println("Using VirtualBox version", version, "executor...")

	if s.Config.VirtualBox.DisableSnapshots {
		s.vmName = s.Config.VirtualBox.BaseName + "-" + s.Build.ProjectUniqueName()
		if vbox.Exist(s.vmName) {
			s.Debugln("Deleting old VM...")
			vbox.Stop(s.vmName)
			vbox.Delete(s.vmName)
			vbox.Unregister(s.vmName)
		}
	} else {
		s.vmName = fmt.Sprintf("%s-runner-%s-concurrent-%d",
			s.Config.VirtualBox.BaseName,
			s.Build.Runner.ShortDescription(),
			s.Build.RunnerID)
	}

	if vbox.Exist(s.vmName) {
		s.Println("Restoring VM from snapshot...")
		err := s.restoreFromSnapshot()
		if err != nil {
			s.Println("Previous VM failed. Deleting, because", err)
			vbox.Stop(s.vmName)
			vbox.Delete(s.vmName)
			vbox.Unregister(s.vmName)
		}
	}

	if !vbox.Exist(s.vmName) {
		s.Println("Creating new VM...")
		err := s.createVM(s.vmName)
		if err != nil {
			return err
		}

		if !s.Config.VirtualBox.DisableSnapshots {
			s.Println("Creating default snapshot...")
			err = vbox.CreateSnapshot(s.vmName, "Started")
			if err != nil {
				return err
			}
		}
	}

	s.Debugln("Checking VM status...")
	status, err := vbox.Status(s.vmName)
	if err != nil {
		return err
	}

	if !vbox.IsStatusOnlineOrTransient(status) {
		s.Println("Starting VM...")
		err := vbox.Start(s.vmName)
		if err != nil {
			return err
		}
	}

	if status != vbox.Running {
		s.Debugln("Waiting for VM to run...")
		err = vbox.WaitForStatus(s.vmName, vbox.Running, 60)
		if err != nil {
			return err
		}
	}

	s.Debugln("Identify SSH Port...")
	sshPort, err := vbox.FindSSHPort(s.vmName)
	s.sshPort = sshPort
	if err != nil {
		return err
	}

	s.Println("Waiting VM to become responsive...")
	err = s.verifyMachine(s.vmName, s.sshPort)
	if err != nil {
		return err
	}

	s.provisioned = true

	s.Println("Starting SSH command...")
	s.sshCommand = ssh.Client{
		Config: *s.Config.SSH,
		Stdout: s.BuildTrace,
		Stderr: s.BuildTrace,
	}
	s.sshCommand.Port = s.sshPort
	s.sshCommand.Host = "localhost"

	s.Debugln("Connecting to SSH server...")
	err = s.sshCommand.Connect()
	if err != nil {
		return err
	}
	return nil
}

func (s *executor) Run(cmd common.ExecutorCommand) error {
	err := s.sshCommand.Run(ssh.Command{
		Environment: s.BuildShell.Environment,
		Command:     s.BuildShell.GetCommandWithArguments(),
		Stdin:       cmd.Script,
		Abort:       cmd.Abort,
	})
	if _, ok := err.(*ssh.ExitError); ok {
		err = &common.BuildError{Inner: err}
	}
	return err
}

func (s *executor) Cleanup() {
	s.sshCommand.Cleanup()

	if s.vmName != "" {
		vbox.Kill(s.vmName)

		if s.Config.VirtualBox.DisableSnapshots || !s.provisioned {
			vbox.Delete(s.vmName)
		}
	}
}

func init() {
	options := executors.ExecutorOptions{
		DefaultBuildsDir: "builds",
		SharedBuildsDir:  false,
		Shell: common.ShellScriptInfo{
			Shell:         "bash",
			Type:          common.LoginShell,
			RunnerCommand: "gitlab-runner",
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

	common.RegisterExecutor("virtualbox", executors.DefaultExecutorProvider{
		Creator:         creator,
		FeaturesUpdater: featuresUpdater,
	})
}
