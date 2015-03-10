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
	ShowHostname     bool
	Config           *common.RunnerConfig
	Build            *common.Build
	BuildsDir        string
	BuildAbort       chan bool
	BuildLogFinish   chan bool
	BuildFinish      chan error
	BuildScript      []byte
	BuildLog         *os.File
}

func (e *AbstractExecutor) FinishBuild(config common.RunnerConfig, buildState common.BuildState, extraMessage string) {
	var buildLog []byte
	if e.BuildLog != nil {
		buildLog, _ = ioutil.ReadFile(e.BuildLog.Name())
	}

	go func() {
		for {
			buffer := io.MultiReader(bytes.NewReader(buildLog), bytes.NewBufferString(extraMessage))
			if common.UpdateBuild(config, e.Build.Id, buildState, buffer) != common.UpdateFailed {
				break
			} else {
				time.Sleep(common.UPDATE_RETRY_INTERVAL * time.Second)
			}
		}

		e.Println("Build finished.")
	}()
}

func (e *AbstractExecutor) WatchTrace(config common.RunnerConfig, abort chan bool, finished chan bool) {
	for {
		select {
		case <-time.After(common.UPDATE_INTERVAL * time.Second):
			if e.BuildLog == nil {
				<-finished
				return
			}

			file, err := os.Open(e.BuildLog.Name())
			if err != nil {
				continue
			}
			defer file.Close()

			switch common.UpdateBuild(config, e.Build.Id, common.Running, file) {
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
	args = append([]interface{}{e.Config.ShortDescription(), e.Build.Id}, args...)
	log.Debugln(args...)
}

func (e *AbstractExecutor) Println(args ...interface{}) {
	if e.BuildLog != nil {
		e.BuildLog.WriteString(fmt.Sprintln(args...))
	}

	args = append([]interface{}{e.Config.ShortDescription(), e.Build.Id}, args...)
	log.Println(args...)
}

func (e *AbstractExecutor) Errorln(args ...interface{}) {
	// write to log file
	if e.BuildLog != nil {
		e.BuildLog.WriteString(fmt.Sprintln(args...))
	}

	args = append([]interface{}{e.Config.ShortDescription(), e.Build.Id}, args...)
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

	var hostname string
	if e.ShowHostname {
		hostname, _ = os.Hostname()
	}

	// Generate build script
	e.BuildsDir = e.DefaultBuildsDir
	if len(e.Config.BuildsDir) != 0 {
		e.BuildsDir = e.Config.BuildsDir
	}

	script, err := e.Build.Generate(e.BuildsDir, hostname)
	if err != nil {
		return err
	}
	e.BuildScript = script

	// Create build log
	build_log, err := ioutil.TempFile("", "build_log")
	if err != nil {
		return err
	}
	e.BuildLog = build_log
	e.Debugln("Created build log:", build_log.Name())
	return nil
}

func (e *AbstractExecutor) Cleanup() {
	if e.BuildLog != nil {
		os.Remove(e.BuildLog.Name())
		e.BuildLog.Close()
	}
}

func (e *AbstractExecutor) Wait() error {
	var buildState common.BuildState
	var buildMessage string

	go e.WatchTrace(*e.Config, e.BuildAbort, e.BuildLogFinish)

	buildTimeout := e.Build.Timeout
	if buildTimeout <= 0 {
		buildTimeout = common.DEFAULT_TIMEOUT
	}

	// Wait for signals: abort, timeout or finish
	log.Debugln(e.Config.ShortDescription(), e.Build.Id, "Waiting for signals...")
	select {
	case <-e.BuildAbort:
		log.Println(e.Config.ShortDescription(), e.Build.Id, "Build got aborted.")
		buildState = common.Failed

	case <-time.After(time.Duration(buildTimeout) * time.Second):
		log.Println(e.Config.ShortDescription(), e.Build.Id, "Build timedout.")
		buildState = common.Failed
		buildMessage = fmt.Sprintf("\nCI Timeout. Execution took longer then %d seconds", buildTimeout)

	case err := <-e.BuildFinish:
		// command finished
		if err != nil {
			log.Println(e.Config.ShortDescription(), e.Build.Id, "Build failed with", err)
			buildState = common.Failed
			buildMessage = fmt.Sprintf("\nBuild failed with %v", err)
		} else {
			log.Println(e.Config.ShortDescription(), e.Build.Id, "Build succeeded.")
			buildState = common.Success
		}
	}

	e.Build.BuildState = buildState
	e.Build.BuildFinished = time.Now()
	e.Build.BuildDuration = e.Build.BuildFinished.Sub(e.Build.BuildStarted)
	e.Debugln("Build took", e.Build.BuildDuration)

	// wait for update log routine to finish
	e.Debugln("Waiting for build log updater to finish")
	e.BuildLogFinish <- true
	e.Debugln("Build log updater finished.")

	// Send final build state to server
	e.FinishBuild(*e.Config, buildState, buildMessage)
	return nil
}
