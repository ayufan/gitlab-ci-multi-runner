package executors

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type ExecutorOptions struct {
	DefaultBuildsDir string
	DefaultCacheDir  string
	SharedBuildsDir  bool
	Shell            common.ShellScriptInfo
	ShowHostname     bool
	SupportedOptions []string
}

type AbstractExecutor struct {
	common.Executor
	ExecutorOptions
	Config      common.RunnerConfig
	Build       *common.Build
	BuildLog    common.BuildTrace
	BuildScript *common.ShellScript
}

func (e *AbstractExecutor) updateShell() error {
	script := &e.Shell
	script.Build = e.Build
	if e.Config.Shell != "" {
		script.Shell = e.Config.Shell
	}
	return nil
}

func (e *AbstractExecutor) generateShellScript() error {
	shellScript, err := common.GenerateShellScript(e.Shell)
	if err != nil {
		return err
	}
	e.BuildScript = shellScript
	e.Debugln("Shell script:", shellScript)
	return nil
}

func (e *AbstractExecutor) startBuild() error {
	// Save hostname
	if e.ShowHostname && e.Build.Hostname == "" {
		e.Build.Hostname, _ = os.Hostname()
	}

	// Start actual build
	rootDir := e.Config.BuildsDir
	if rootDir == "" {
		rootDir = e.DefaultBuildsDir
	}
	cacheDir := e.Config.CacheDir
	if cacheDir == "" {
		cacheDir = e.DefaultCacheDir
	}
	e.Build.StartBuild(rootDir, cacheDir, e.SharedBuildsDir)
	return nil
}

func (e *AbstractExecutor) verifyOptions() error {
	supportedOptions := e.SupportedOptions
	if shell := common.GetShell(e.Shell.Shell); shell != nil {
		supportedOptions = append(supportedOptions, shell.GetSupportedOptions()...)
	}

	for key, value := range e.Build.Options {
		if value == nil {
			continue
		}
		found := false
		for _, option := range supportedOptions {
			if option == key {
				found = true
				break
			}
		}

		if !found {
			e.Warningln(key, "is not supported by selected executor and shell")
		}
	}
	return nil
}

func (e *AbstractExecutor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	e.Config = *config
	e.Build = build
	e.BuildLog = build.Trace

	err := e.startBuild()
	if err != nil {
		return err
	}

	e.Infoln(fmt.Sprintf("%s %s (%s)", common.NAME, common.VERSION, common.REVISION))

	err = e.updateShell()
	if err != nil {
		return err
	}

	err = e.verifyOptions()
	if err != nil {
		return err
	}

	err = e.generateShellScript()
	if err != nil {
		return err
	}
	return nil
}

func (e *AbstractExecutor) runScript(abort chan interface{}) error {
	// Execute pre script (git clone, cache restore, artifacts download)
	err := e.Run(common.ExecutorCommand{
		Script:     e.BuildScript.PreScript,
		Predefined: true,
		Abort:      abort,
	})

	if err == nil {
		// Execute build script (user commands)
		err = e.Run(common.ExecutorCommand{
			Script:     e.BuildScript.BuildScript,
			Abort:      abort,
		})

		// Execute after script (user commands)
		if e.BuildScript.AfterScript != "" {
			timeoutCh := make(chan interface{})
			go func() {
				timeoutCh <- <- time.After(time.Minute * 5)
			}()
			e.Run(common.ExecutorCommand{
				Script:     e.BuildScript.AfterScript,
				Abort:      timeoutCh,
			})
		}
	}

	// Execute post script (cache store, artifacts upload)
	if err == nil {
		err = e.Run(common.ExecutorCommand{
			Script:     e.BuildScript.PostScript,
			Predefined: true,
			Abort:      abort,
		})
	}

	return err
}

func (e *AbstractExecutor) Wait() (err error) {
	buildTimeout := e.Build.Timeout
	if buildTimeout <= 0 {
		buildTimeout = common.DefaultTimeout
	}

	buildCanceled := make(chan bool)
	buildFinish := make(chan error)
	buildAbort := make(chan interface{})

	// Wait for cancel notification
	e.Build.Trace.Notify(func() {
		buildCanceled <- true
	})

	// Run build script
	go func() {
		buildFinish <- e.runScript(buildAbort)
	}()

	// Wait for signals: cancel, timeout, abort or finish
	e.Debugln("Waiting for signals...")
	select {
	case <-buildCanceled:
		err = errors.New("canceled")

	case <-time.After(time.Duration(buildTimeout) * time.Second):
		err = fmt.Errorf("execution took longer than %v seconds", buildTimeout)

	case signal := <-e.Build.BuildAbort:
		err = fmt.Errorf("aborted: %v", signal)

	case err = <-buildFinish:
		return err
	}

	// Wait till we receive that build did finish
	for {
		select {
		case buildAbort <- true:
		case <-buildFinish:
			return err
		}
	}
}

func (e *AbstractExecutor) Finish(err error) {
	if err != nil {
		e.Println()
		e.Errorln("Build failed:", err)
	} else {
		e.Println()
		e.Infoln("Build succeeded")
	}
}

func (e *AbstractExecutor) Cleanup() {
}
