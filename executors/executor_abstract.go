package executors

import (
	"bytes"
	"fmt"
	"io"
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
	BuildsDir        string
	BuildAbort       chan bool
	BuildLogFinish   chan bool
	BuildFinish      chan error
	BuildLog         *os.File
	ShellScript      *common.ShellScript
}

func (e *AbstractExecutor) FinishBuild(config common.RunnerConfig, buildState common.BuildState, extraMessage string) {
	var buildLog []byte
	if e.BuildLog != nil {
		buildLog, _ = ioutil.ReadFile(e.BuildLog.Name())
	}

	for {
		buffer := io.MultiReader(bytes.NewReader(buildLog), bytes.NewBufferString(extraMessage))
		if common.UpdateBuild(config, e.Build.ID, buildState, buffer) != common.UpdateFailed {
			break
		} else {
			time.Sleep(common.UpdateRetryInterval * time.Second)
		}
	}

	e.Println("Build finished.")
}

func (e *AbstractExecutor) WatchTrace(config common.RunnerConfig, abort chan bool, finished chan bool) {
	for {
		select {
		case <-time.After(common.UpdateInterval * time.Second):
			if e.BuildLog == nil {
				<-finished
				return
			}

			file, err := os.Open(e.BuildLog.Name())
			if err != nil {
				continue
			}
			defer file.Close()

			switch common.UpdateBuild(config, e.Build.ID, common.Running, file) {
			case common.UpdateSucceeded:
			case common.UpdateAbort:
				e.Debugln("updateBuildLog", "Sending abort request...")
				abort <- true
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

func (e *AbstractExecutor) Prepare(config *common.RunnerConfig, build *common.Build) error {
	e.Config = config
	e.Build = build
	e.BuildAbort = make(chan bool, 1)
	e.BuildFinish = make(chan error, 1)
	e.BuildLogFinish = make(chan bool)
	build.BuildStarted = time.Now()
	build.BuildState = common.Pending

	e.Println("Starting build...")

	if e.ShowHostname {
		build.Hostname, _ = os.Hostname()
	}

	// Generate build script
	e.BuildsDir = e.DefaultBuildsDir
	if len(e.Config.BuildsDir) != 0 {
		e.BuildsDir = e.Config.BuildsDir
	}
	build.BuildsDir = e.BuildsDir

	shell := e.DefaultShell
	if e.Config.Shell != "" {
		shell = e.Config.Shell
	}

	shellScript, err := common.GenerateShellScript(shell, build)
	if err != nil {
		return err
	}
	e.ShellScript = shellScript

	// Create build log
	buildLog, err := ioutil.TempFile("", "build_log")
	if err != nil {
		return err
	}
	e.BuildLog = buildLog
	e.Debugln("Created build log:", buildLog.Name())
	go e.WatchTrace(*e.Config, e.BuildAbort, e.BuildLogFinish)
	return nil
}

func (e *AbstractExecutor) Wait() error {
	e.Build.BuildState = common.Running

	buildTimeout := e.Build.Timeout
	if buildTimeout <= 0 {
		buildTimeout = common.DefaultTimeout
	}

	// Wait for signals: abort, timeout or finish
	log.Debugln(e.Config.ShortDescription(), e.Build.ID, "Waiting for signals...")
	select {
	case <-e.BuildAbort:
		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build got aborted.")
		e.Build.BuildState = common.Failed
		e.Build.BuildMessage = "\nBuild got aborted"

	case <-time.After(time.Duration(buildTimeout) * time.Second):
		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build timedout.")
		e.Build.BuildState = common.Failed
		e.Build.BuildMessage = fmt.Sprintf("\nCI Timeout. Execution took longer then %d seconds", buildTimeout)

	case signal := <-e.Build.BuildAbort:
		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build got aborted", signal)
		e.Build.BuildState = common.Failed
		e.Build.BuildMessage = fmt.Sprintf("\nBuild got aborted: %v", signal)

	case err := <-e.BuildFinish:
		if err != nil {
			return err
		}

		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build succeeded.")
		e.Build.BuildState = common.Success
		e.Build.BuildMessage = "\n"
	}
	return nil
}

func (e *AbstractExecutor) Finish(err error) {
	if err != nil {
		e.Build.BuildState = common.Failed
		e.Build.BuildMessage = fmt.Sprintf("\nBuild failed with %v", err)
	}

	e.Build.BuildFinished = time.Now()
	e.Build.BuildDuration = e.Build.BuildFinished.Sub(e.Build.BuildStarted)
	e.Debugln("Build took", e.Build.BuildDuration)

	if e.BuildLog != nil {
		// wait for update log routine to finish
		e.Debugln("Waiting for build log updater to finish")
		e.BuildLogFinish <- true
		e.Debugln("Build log updater finished.")
	}

	// Send final build state to server
	e.FinishBuild(*e.Config, e.Build.BuildState, e.Build.BuildMessage)
}

func (e *AbstractExecutor) Cleanup() {
	if e.BuildLog != nil {
		os.Remove(e.BuildLog.Name())
		e.BuildLog.Close()
	}
}
