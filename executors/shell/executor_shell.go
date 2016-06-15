package shell

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/kardianos/osext"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"time"
)

type executor struct {
	executors.AbstractExecutor
}

func (s *executor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	if globalConfig != nil {
		s.Shell().User = globalConfig.User
	}

	// expand environment variables to have current directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Getwd: %v", err)
	}

	mapping := func(key string) string {
		switch key {
		case "PWD":
			return wd
		default:
			return ""
		}
	}

	s.DefaultBuildsDir = os.Expand(s.DefaultBuildsDir, mapping)
	s.DefaultCacheDir = os.Expand(s.DefaultCacheDir, mapping)

	// Pass control to executor
	err = s.AbstractExecutor.Prepare(globalConfig, config, build)
	if err != nil {
		return err
	}

	s.Println("Using Shell executor...")
	return nil
}

func (s *executor) killAndWait(cmd *exec.Cmd, waitCh chan error) error {
	for {
		s.Debugln("Aborting command...")
		helpers.KillProcessGroup(cmd)
		select {
		case <-time.After(time.Second):
		case err := <-waitCh:
			return err
		}
	}
}

func (s *executor) Run(cmd common.ExecutorCommand) error {
	// Create execution command
	c := exec.Command(s.BuildShell.Command, s.BuildShell.Arguments...)
	if c == nil {
		return errors.New("Failed to generate execution command")
	}

	helpers.SetProcessGroup(c)
	defer helpers.KillProcessGroup(c)

	// Fill process environment variables
	c.Env = append(os.Environ(), s.BuildShell.Environment...)
	c.Stdout = s.BuildLog
	c.Stderr = s.BuildLog

	if s.BuildShell.PassFile {
		scriptDir, err := ioutil.TempDir("", "build_script")
		if err != nil {
			return err
		}
		defer os.RemoveAll(scriptDir)

		scriptFile := filepath.Join(scriptDir, "script."+s.BuildShell.Extension)
		err = ioutil.WriteFile(scriptFile, []byte(cmd.Script), 0700)
		if err != nil {
			return err
		}

		c.Args = append(c.Args, scriptFile)
	} else {
		c.Stdin = bytes.NewBufferString(cmd.Script)
	}

	// Start a process
	err := c.Start()
	if err != nil {
		return fmt.Errorf("Failed to start process: %s", err)
	}

	// Wait for process to finish
	waitCh := make(chan error)
	go func() {
		waitCh <- c.Wait()
	}()

	// Support process abort
	select {
	case err = <-waitCh:
		return err

	case <-cmd.Abort:
		return s.killAndWait(c, waitCh)
	}
}

func init() {
	// Look for self
	runnerCommand, err := osext.Executable()
	if err != nil {
		logrus.Warningln(err)
	}

	options := executors.ExecutorOptions{
		DefaultBuildsDir: "$PWD/builds",
		DefaultCacheDir:  "$PWD/cache",
		SharedBuildsDir:  true,
		Shell: common.ShellScriptInfo{
			Shell:         common.GetDefaultShell(),
			Type:          common.LoginShell,
			RunnerCommand: runnerCommand,
		},
		ShowHostname: false,
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

	common.RegisterExecutor("shell", executors.DefaultExecutorProvider{
		Creator:         creator,
		FeaturesUpdater: featuresUpdater,
	})
}
