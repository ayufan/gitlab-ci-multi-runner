package shell

import (
	"bytes"
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
	cmd       []*exec.Cmd
	scriptDir string
}

func (s *executor) Prepare(build *common.Build, data common.ExecutorData) error {
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
	err = s.AbstractExecutor.Prepare(build, data)
	if err != nil {
		return err
	}

	s.Println("Using Shell executor...")
	return nil
}

func (s *executor) Run(run common.ExecutorRun) (err error) {
	cmd := exec.Command(run.Command, run.Arguments...)
	helpers.SetProcessGroup(cmd)
	cmd.Env = append(os.Environ(), run.Environment...)
	cmd.Stdout = run.Trace
	cmd.Stderr = run.Trace

	if run.Extension != "" {
		scriptDir, err := ioutil.TempDir("", "build_script")
		if err != nil {
			return err
		}
		defer os.RemoveAll(s.scriptDir)

		scriptFile := filepath.Join(scriptDir, "script."+run.Extension)
		err = ioutil.WriteFile(scriptFile, []byte(run.Script), 0700)
		if err != nil {
			return err
		}

		cmd.Args = append(cmd.Args, scriptFile)
	} else {
		cmd.Stdin = bytes.NewReader([]byte(run.Script))
	}

	// Start process
	err = cmd.Start()
	if err != nil {
		return
	}
	defer helpers.KillProcessGroup(cmd)

	// Asynchronously wait for result
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- cmd.Wait()
	}()

	// Wait for process to finish
	select {
	case err = <-resultCh:
		return
	case err = <-run.Abort:
		return
	}
}

func (s *executor) Cleanup() {
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
		Shell:            common.GetDefaultShell(),
		Type:             common.LoginShell,
		RunnerCommand:    runnerCommand,
		ShowHostname:     false,
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
