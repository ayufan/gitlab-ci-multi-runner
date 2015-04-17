package executors

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ayufan/gitlab-ci-multi-runner/common"
)

type AbstractExecutor struct {
	DefaultBuildsDir string
	DefaultShell     string
	ShowHostname     bool
	Config           *common.RunnerConfig
	Build            *common.Build
	BuildCanceled    chan bool
	FinishLogWatcher chan bool
	BuildFinish      chan error
	BuildLog         *os.File
	ShellScript      *common.ShellScript
}

func (e *AbstractExecutor) WatchTrace(config common.RunnerConfig, canceled chan bool, finished chan bool) {
	buildLog := e.BuildLog
	if buildLog == nil {
		<-finished
		return
	}

	for {
		select {
		case <-time.After(common.UpdateInterval * time.Second):
			buildTrace, err := e.Build.ReadBuildLog()
			if err != nil {
				e.Debugln("updateBuildLog", "Failed to read build log...", err)
				continue
			}

			switch common.UpdateBuild(config, e.Build.ID, common.Running, buildTrace) {
			case common.UpdateSucceeded:
			case common.UpdateAbort:
				e.Debugln("updateBuildLog", "Sending abort request...")
				canceled <- true
				e.Debugln("updateBuildLog", "Waiting for finished flag...")
				<-finished
				e.Debugln("updateBuildLog", "Thread finished.")
				return
			case common.UpdateFailed:
			}

		case <-finished:
			e.Debugln("updateBuildLog", "Received finish.")
			return
		}
	}
}

func (e *AbstractExecutor) Debugln(args ...interface{}) {
	args = append([]interface{}{e.Config.ShortDescription(), e.Build.ID}, args...)
	log.Debugln(args...)
}

func (e *AbstractExecutor) Println(args ...interface{}) {
	if e.BuildLog != nil {
		e.BuildLog.WriteString(fmt.Sprintln(args...))
	}

	args = append([]interface{}{e.Config.ShortDescription(), e.Build.ID}, args...)
	log.Println(args...)
}

func (e *AbstractExecutor) Errorln(args ...interface{}) {
	// write to log file
	if e.BuildLog != nil {
		e.BuildLog.WriteString(fmt.Sprintln(args...))
	}

	args = append([]interface{}{e.Config.ShortDescription(), e.Build.ID}, args...)
	log.Errorln(args...)
}

func (e *AbstractExecutor) generateShellScript() error {
	shell := e.DefaultShell
	if e.Config.Shell != "" {
		shell = e.Config.Shell
	}

	shellScript, err := common.GenerateShellScript(shell, e.Build)
	if err != nil {
		return err
	}
	e.ShellScript = shellScript
	e.Debugln("Shell script:", shellScript)
	return nil
}

func (e *AbstractExecutor) startBuild() error {
	buildsDir := e.DefaultBuildsDir
	if e.Config.BuildsDir != "" {
		buildsDir = e.Config.BuildsDir
	}

	buildLog, err := ioutil.TempFile("", "build_log")
	if err != nil {
		return err
	}
	e.BuildLog = buildLog
	e.Debugln("Created build log:", buildLog.Name())

	// Save hostname
	if e.ShowHostname {
		e.Build.Hostname, _ = os.Hostname()
	}

	// Start actual build
	e.Build.StartBuild(buildsDir, buildLog.Name())
	return nil
}

func (e *AbstractExecutor) Prepare(config *common.RunnerConfig, build *common.Build) error {
	e.Config = config
	e.Build = build
	e.BuildCanceled = make(chan bool, 1)
	e.BuildFinish = make(chan error, 1)
	e.FinishLogWatcher = make(chan bool)
	e.Println("Starting build...")

	err := e.startBuild()
	if err != nil {
		return err
	}

	err = e.generateShellScript()
	if err != nil {
		return err
	}

	go e.WatchTrace(*e.Config, e.BuildCanceled, e.FinishLogWatcher)
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
	case <-e.BuildCanceled:
		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build got canceled.")
		e.Build.FinishBuild(common.Failed, "Build got canceled")

	case <-time.After(time.Duration(buildTimeout) * time.Second):
		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build timedout.")
		e.Build.FinishBuild(common.Failed, "CI Timeout. Execution took longer then %d seconds", buildTimeout)

	case signal := <-e.Build.BuildAbort:
		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build got aborted", signal)
		e.Build.FinishBuild(common.Failed, "Build got aborted: %v", signal)

	case err := <-e.BuildFinish:
		if err != nil {
			return err
		}

		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build succeeded.")
		e.Build.FinishBuild(common.Success, "Build succeeded.")
	}
	return nil
}

func (e *AbstractExecutor) Finish(err error) {
	if err != nil {
		e.Build.FinishBuild(common.Failed, "Build failed with %v", err)
	}

	e.Debugln("Build took", e.Build.BuildDuration)

	if e.BuildLog != nil {
		// wait for update log routine to finish
		e.Debugln("Waiting for build log updater to finish")
		e.FinishLogWatcher <- true
		e.Debugln("Build log updater finished.")
	}

	// Send final build state to server
	e.Build.SendBuildLog()
	e.Println("Build finished.")
}

func (e *AbstractExecutor) Cleanup() {
	if e.BuildLog != nil {
		os.Remove(e.BuildLog.Name())
		e.BuildLog.Close()
	}
}
