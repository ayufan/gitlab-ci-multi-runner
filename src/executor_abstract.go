package src

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
)

type AbstractExecutor struct {
	DefaultBuildsDir string
	ShowHostname     bool
	config           *RunnerConfig
	build            *Build
	builds_dir       string
	buildAbort       chan bool
	buildLogFinish   chan bool
	buildFinish      chan error
	BuildScript      []byte
	BuildLog         *os.File
}

func (e *AbstractExecutor) FinishBuild(config RunnerConfig, buildState BuildState, extraMessage string) {
	var buildLog []byte
	if e.BuildLog != nil {
		buildLog, _ = ioutil.ReadFile(e.BuildLog.Name())
	}

	go func() {
		for {
			buffer := io.MultiReader(bytes.NewReader(buildLog), bytes.NewBufferString(extraMessage))
			if UpdateBuild(config, e.build.Id, buildState, buffer) != UpdateFailed {
				break
			} else {
				time.Sleep(UPDATE_RETRY_INTERVAL * time.Second)
			}
		}

		e.println("Build finished.")
	}()
}

func (e *AbstractExecutor) WatchTrace(config RunnerConfig, abort chan bool, finished chan bool) {
	for {
		select {
		case <-time.After(UPDATE_INTERVAL * time.Second):
			if e.BuildLog == nil {
				<-finished
				return
			}

			file, err := os.Open(e.BuildLog.Name())
			if err != nil {
				continue
			}
			defer file.Close()

			switch UpdateBuild(config, e.build.Id, Running, file) {
			case UpdateSucceeded:
			case UpdateAbort:
				e.debugln("updateBuildLog", "Sending abort request...")
				abort <- true
				e.debugln("updateBuildLog", "Waiting for finished flag...")
				<-finished
				e.debugln("updateBuildLog", "Thread finished.")
				return
			case UpdateFailed:
			}

		case <-finished:
			e.debugln("updateBuildLog", "Received finish.")
			return
		}
	}
}

func (e *AbstractExecutor) debugln(args ...interface{}) {
	args = append([]interface{}{e.config.ShortDescription(), e.build.Id}, args...)
	log.Debugln(args...)
}

func (e *AbstractExecutor) println(args ...interface{}) {
	if e.BuildLog != nil {
		e.BuildLog.WriteString(fmt.Sprintln(args...))
	}

	args = append([]interface{}{e.config.ShortDescription(), e.build.Id}, args...)
	log.Println(args...)
}

func (e *AbstractExecutor) errorln(args ...interface{}) {
	// write to log file
	if e.BuildLog != nil {
		e.BuildLog.WriteString(fmt.Sprintln(args...))
	}

	args = append([]interface{}{e.config.ShortDescription(), e.build.Id}, args...)
	log.Errorln(args...)
}

func (e *AbstractExecutor) Prepare(config *RunnerConfig, build *Build) error {
	e.config = config
	e.build = build
	e.buildAbort = make(chan bool, 1)
	e.buildFinish = make(chan error, 1)
	e.buildLogFinish = make(chan bool)
	build.BuildStarted = time.Now()
	build.BuildState = Pending

	e.println("Starting build...")

	var hostname string
	if e.ShowHostname {
		hostname, _ = os.Hostname()
	}

	// Generate build script
	e.builds_dir = e.DefaultBuildsDir
	if len(e.config.BuildsDir) != 0 {
		e.builds_dir = e.config.BuildsDir
	}

	script, err := e.build.Generate(e.builds_dir, hostname)
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
	e.debugln("Created build log:", build_log.Name())
	return nil
}

func (e *AbstractExecutor) Cleanup() {
	if e.BuildLog != nil {
		os.Remove(e.BuildLog.Name())
		e.BuildLog.Close()
	}
}

func (e *AbstractExecutor) Wait() error {
	var buildState BuildState
	var buildMessage string

	go e.WatchTrace(*e.config, e.buildAbort, e.buildLogFinish)

	buildTimeout := e.build.Timeout
	if buildTimeout <= 0 {
		buildTimeout = DEFAULT_TIMEOUT
	}

	// Wait for signals: abort, timeout or finish
	log.Debugln(e.config.ShortDescription(), e.build.Id, "Waiting for signals...")
	select {
	case <-e.buildAbort:
		log.Println(e.config.ShortDescription(), e.build.Id, "Build got aborted.")
		buildState = Failed

	case <-time.After(time.Duration(buildTimeout) * time.Second):
		log.Println(e.config.ShortDescription(), e.build.Id, "Build timedout.")
		buildState = Failed
		buildMessage = fmt.Sprintf("\nCI Timeout. Execution took longer then %d seconds", buildTimeout)

	case err := <-e.buildFinish:
		// command finished
		if err != nil {
			log.Println(e.config.ShortDescription(), e.build.Id, "Build failed with", err)
			buildState = Failed
			buildMessage = fmt.Sprintf("\nBuild failed with %v", err)
		} else {
			log.Println(e.config.ShortDescription(), e.build.Id, "Build succeeded.")
			buildState = Success
		}
	}

	e.build.BuildState = buildState
	e.build.BuildFinished = time.Now()
	e.build.BuildDuration = e.build.BuildFinished.Sub(e.build.BuildStarted)
	e.println("Build took", e.build.BuildDuration)

	// wait for update log routine to finish
	e.debugln("Waiting for build log updater to finish")
	e.buildLogFinish <- true
	e.debugln("Build log updater finished.")

	// Send final build state to server
	e.FinishBuild(*e.config, buildState, buildMessage)
	return nil
}
