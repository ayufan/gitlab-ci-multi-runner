package executors

import (
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"io"
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
	ExecutorOptions
	Config      common.RunnerConfig
	Build       *common.Build
	BuildFinish chan error
	BuildLog    io.WriteCloser
	BuildScript *common.ShellScript

	buildCanceled     chan bool
	finishUpdateTrace chan bool
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
	// Create pipe where data are read
	if e.Build.Network != nil {
		e.finishUpdateTrace = make(chan bool)
		reader, writer := io.Pipe()
		e.BuildLog = writer
		go e.readTrace(reader)
		go e.updateTrace(e.Config, e.buildCanceled, e.finishUpdateTrace)
	} else {
		e.BuildLog = os.Stdout
	}

	// Save hostname
	if e.ShowHostname {
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
	e.buildCanceled = make(chan bool, 1)
	e.BuildFinish = make(chan error, 1)

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

func (e *AbstractExecutor) Wait() error {
	e.Build.BuildState = common.Running

	buildTimeout := e.Build.Timeout
	if buildTimeout <= 0 {
		buildTimeout = common.DefaultTimeout
	}

	// Wait for signals: cancel, timeout, abort or finish
	log.Debugln(e.Config.ShortDescription(), e.Build.ID, "Waiting for signals...")
	select {
	case <-e.buildCanceled:
		e.Println()
		e.Warningln("Build got canceled.")
		e.Build.FinishBuild(common.Failed)

	case <-time.After(time.Duration(buildTimeout) * time.Second):
		e.Println()
		e.Errorln("CI Timeout. Execution took longer then", buildTimeout, "seconds.")
		e.Build.FinishBuild(common.Failed)

	case signal := <-e.Build.BuildAbort:
		e.Println()
		e.Errorln("Build got aborted:", signal)
		e.Build.FinishBuild(common.Failed)

	case err := <-e.BuildFinish:
		if err != nil {
			return err
		}

		e.Println()
		e.Infoln("Build succeeded.")
		e.Build.FinishBuild(common.Success)
	}
	return nil
}

func (e *AbstractExecutor) Finish(err error) {
	if err != nil {
		e.Println()
		e.Errorln("Build failed with:", err)
		e.Build.FinishBuild(common.Failed)
	}

	e.Debugln("Build took", e.Build.BuildDuration)

	if e.finishUpdateTrace != nil {
		// wait for update log routine to finish
		e.Debugln("Waiting for build log updater to finish")
		e.finishUpdateTrace <- true
		e.Debugln("Build log updater finished.")
	}

	// Send final build state to server
	if e.Build.Network != nil {
		e.Debugln("Build log: ", e.Build.BuildLog())
		e.Build.SendBuildLog()
	}
	e.Debugln("Build finished")
}

func (e *AbstractExecutor) Cleanup() {
	if e.BuildLog != nil {
		e.BuildLog.Close()
	}
	e.Debugln("Cleanup finished")
}
