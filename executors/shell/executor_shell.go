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
)

type executor struct {
	executors.AbstractExecutor
	cmd       *exec.Cmd
	scriptDir string
}

func (s *executor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	if globalConfig != nil {
		s.Shell.User = globalConfig.User
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

func (s *executor) Start() error {
	s.Debugln("Starting shell command...")

	// Create execution command
	s.cmd = exec.Command(s.BuildScript.Command, s.BuildScript.Arguments...)
	if s.cmd == nil {
		return errors.New("Failed to generate execution command")
	}

	helpers.SetProcessGroup(s.cmd)

	// Fill process environment variables
	s.cmd.Env = append(os.Environ(), s.BuildScript.Environment...)
	s.cmd.Stdout = s.BuildLog
	s.cmd.Stderr = s.BuildLog

	if s.BuildScript.PassFile {
		scriptDir, err := ioutil.TempDir("", "build_script")
		if err != nil {
			return err
		}
		s.scriptDir = scriptDir

		scriptFile := filepath.Join(scriptDir, "script."+s.BuildScript.Extension)
		err = ioutil.WriteFile(scriptFile, s.BuildScript.GetScriptBytes(), 0700)
		if err != nil {
			return err
		}

		s.cmd.Args = append(s.cmd.Args, scriptFile)
	} else {
		s.cmd.Stdin = bytes.NewReader(s.BuildScript.GetScriptBytes())
	}

	// Start process
	err := s.cmd.Start()
	if err != nil {
		return fmt.Errorf("Failed to start process: %s", err)
	}

	// Wait for process to exit
	go func() {
		s.BuildFinish <- s.cmd.Wait()
	}()
	return nil
}

func (s *executor) Cleanup() {
	helpers.KillProcessGroup(s.cmd)

	if s.scriptDir != "" {
		os.RemoveAll(s.scriptDir)
	}

	s.AbstractExecutor.Cleanup()
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
