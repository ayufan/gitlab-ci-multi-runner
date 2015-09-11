package vbox

import (
    "errors"
    "fmt"
    "time"

    "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
    "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
    "gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/ssh"

    "gitlab.com/gitlab-org/gitlab-ci-multi-runner/vbox"

    "gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
)

type VboxExecutor struct {
    executors.AbstractExecutor
	// cmd             *exec.Cmd
    vmName          string
    sshCommand      ssh.Command
    sshPort         string
    provisioned     bool
    machineVerified bool
}

func (s *VboxExecutor) verifyMachine(vmName string, sshPort string) error {
	if s.machineVerified {
		return nil
	}

	// Create SSH command
	sshCommand := ssh.Command{
		Config:         *s.Config.SSH,
		Command:        "exit 0",
		Stdout:         s.BuildLog,
		Stderr:         s.BuildLog,
		ConnectRetries: 30,
	}
    host := `localhost`
    sshCommand.Port = &sshPort
    sshCommand.Host = &host

	s.Debugln("Connecting to SSH...")
    err := sshCommand.Connect()
	if err != nil {
		return err
	}
	defer sshCommand.Cleanup()
	err = sshCommand.Run()
	if err != nil {
		return err
	}
	s.machineVerified = true
	return nil
}

func (s *VboxExecutor) restoreFromSnapshot() error {
    s.Debugln("Reverting VM to current snapshot...")
    err := vbox.RevertToSnapshot(s.vmName)
    if err != nil {
        return err
    }

    return nil
}

// Vbox doesn't support templates
func (s *VboxExecutor) CreateVM(vmName string) error {
	baseImage := s.Config.Vbox.BaseName
	if baseImage == "" {
		return errors.New("Missing Image setting from Vbox config")
	}

    vmStatus, _ := vbox.Status(vmName)
    if vmStatus == vbox.Invalid {
        vbox.Unregister(vmName)
    }

    if !vbox.Exist(vmName) {
        s.Debugln("Creating testing VM from VM", baseImage, "...")
        err := vbox.CreateOsVM(baseImage, vmName)
        if err != nil {
            return err
        }
    }

    s.Debugln("Creating localhost ssh forwarding...")
    err := vbox.ConfigureSSH(vmName)
    if err != nil {
        return err
    }

    s.Debugln("Bootstraping VM...")
    err = vbox.Start(s.vmName)
    if err != nil {
        return err
    }

    s.Debugln("Identify SSH Port...")
    sshPort, err := vbox.FindSshPort(s.vmName)
    s.sshPort = sshPort
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

func (s *VboxExecutor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
    err:= s.AbstractExecutor.Prepare(globalConfig, config, build)
    if err != nil {
        return err
    }

    if s.ShellScript.PassFile {
        return errors.New("Vbox doesn't support shells that require script file")
    }

    if s.Config.SSH == nil {
        return errors.New("Missing SSH config")
    }

    if s.Config.Vbox == nil {
        return errors.New("Missing Vbox configuration")
    }

    if s.Config.Vbox.BaseName == "" {
        return errors.New("Missing BaseName setting from Vbox config")
    }

    version, err := vbox.Version()
    if err != nil {
        panic(err)
    }

    s.Println("Using VirtualBox version", version, "executor...")

    vmStatus, _ := vbox.Status(s.vmName)
    if vmStatus == vbox.Invalid {
        vbox.Unregister(s.vmName)
    }

    if helpers.BoolOrDefault(s.Config.Vbox.DisableSnapshots, false) {
        s.vmName =s.Config.Vbox.BaseName + "-" + s.Build.ProjectUniqueName()
        if vbox.Exist(s.vmName) {
            s.Debugln("Deleting old VM...")
            vbox.Stop(s.vmName)
            vbox.Delete(s.vmName)
            vbox.Unregister(s.vmName)
        }
    } else {
        s.vmName = fmt.Sprintf("%s-runner-%s-concurrent-%d",
            s.Config.Vbox.BaseName,
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
        err := s.CreateVM(s.vmName)
        if err != nil {
            return err
        }

        if !helpers.BoolOrDefault(s.Config.Vbox.DisableSnapshots, false) {
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

    if status == vbox.Stopped || status == vbox.Suspended || status == vbox.Saved {
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
    sshPort, err := vbox.FindSshPort(s.vmName)
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

    return nil
}

func (s *VboxExecutor) Start() error {
    s.Println("Starting SSH command...")
    s.sshCommand = ssh.Command{
		Config:      *s.Config.SSH,
		Environment: s.ShellScript.Environment,
		Command:     s.ShellScript.GetFullCommand(),
		Stdin:       s.ShellScript.GetScriptBytes(),
		Stdout:      s.BuildLog,
		Stderr:      s.BuildLog,
	}
    host := `localhost`
	s.sshCommand.Port = &s.sshPort
    s.sshCommand.Host = &host

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

func (s *VboxExecutor) Cleanup() {
    s.sshCommand.Cleanup()

    if s.vmName != "" {
        vbox.Kill(s.vmName)

        if helpers.BoolOrDefault(s.Config.Vbox.DisableSnapshots, false) || !s.provisioned {
            vbox.Delete(s.vmName)
        }
    }
}

func init() {
    options := executors.ExecutorOptions{
        DefaultBuildsDir: "builds",
        SharedBuildsDir: false,
        Shell: common.ShellScriptInfo{
            Shell: "bash",
            Type:  common.LoginShell,
        },
        ShowHostname: true,
    }

    create := func() common.Executor {
		return &VboxExecutor{
			AbstractExecutor: executors.AbstractExecutor{
				ExecutorOptions: options,
			},
		}
    }

    common.RegisterExecutor("vbox", common.ExecutorFactory{
        Create: create,
        Features: common.FeaturesInfo{
                Variables: true,
        },
    })
}
