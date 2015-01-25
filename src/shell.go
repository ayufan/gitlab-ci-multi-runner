package src

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	log "github.com/Sirupsen/logrus"
)

type ShellExecutor struct {
}

func sendBuildLog(config RunnerConfig, buildId int, build_log string, state BuildState, extraMessage string) UpdateState {
	file, err := os.Open(build_log)
	if err != nil {
		return UpdateBuild(config, buildId, state, bytes.NewBufferString(""))
	}
	defer file.Close()

	buffer := io.MultiReader(file, bytes.NewBufferString(extraMessage))
	return UpdateBuild(config, buildId, state, buffer)
}

func updateBuildLog(config RunnerConfig, buildId int, build_log string, abort chan bool, finished chan bool) {
	for {
		select {
		case <-time.After(UPDATE_INTERVAL * time.Second):
			log.Debugln(config.ShortDescription(), buildId, "updateBuildLog", "Updating...")
			switch sendBuildLog(config, buildId, build_log, Running, "") {
			case UpdateSucceeded:
			case UpdateAbort:
				log.Debugln(config.ShortDescription(), buildId, "updateBuildLog", "Sending abort request...")
				abort <- true
				log.Debugln(config.ShortDescription(), buildId, "updateBuildLog", "Waiting for finished flag...")
				<-finished
				log.Debugln(config.ShortDescription(), buildId, "updateBuildLog", "Thread finished.")
				return
			case UpdateFailed:
			}

		case <-finished:
			log.Debugln(config.ShortDescription(), buildId, "updateBuildLog", "Received finish.")
			return
		}
	}
}

func (s *ShellExecutor) Run(config RunnerConfig, build Build) error {
	builds_dir := "tmp/builds"

	// generate build script
	script_file := build.Generate(builds_dir)
	if script_file == nil {
		return errors.New("Failed to generate build script")
	}
	defer os.Remove(*script_file)
	log.Debugln(config.ShortDescription(), build.Id, "Generated build script:", *script_file)

	// create build log
	build_log, err := ioutil.TempFile("", "build_log")
	if err != nil {
		return errors.New("Failed to create build log file")
	}
	defer build_log.Close()
	defer os.Remove(build_log.Name())
	log.Debugln(config.ShortDescription(), build.Id, "Created build log:", build_log.Name())

	// create execution command
	cmd := exec.Command("/usr/local/bin/setsid", *script_file)
	if cmd == nil {
		return errors.New("Failed to generate execution command")
	}

	cmd.Env = []string{
		"CI_SERVER=yes",
		"CI_SERVER_NAME=GitLab CI",
		"CI_SERVER_VERSION=",
		"CI_SERVER_REVISION=",

		fmt.Sprintf("CI_BUILD_REF=%s", build.Sha),
		fmt.Sprintf("CI_BUILD_BEFORE_SHA=%s", build.BeforeSha),
		fmt.Sprintf("CI_BUILD_REF_NAME=%s", build.RefName),
		fmt.Sprintf("CI_BUILD_ID=%d", build.Id),
		fmt.Sprintf("CI_BUILD_REPO=%s", build.RepoURL),

		fmt.Sprintf("CI_PROJECT_ID=%d", build.ProjectId),

		"RUBYLIB=",
		"RUBYOPT=",
		"BNDLE_BIN_PATH=",
		"BUNDLE_GEMFILE=",
	}

	// cmd.Stdin = ioutil.
	cmd.Stdout = build_log
	cmd.Stderr = build_log

	// Start process
	err = cmd.Start()
	if err != nil {
		return errors.New("Failed to start process")
	}
	log.Debugln(config.ShortDescription(), build.Id, "Started build process")

	// Wait for process to exit
	command_finish := make(chan error, 1)
	go func() {
		command_finish <- cmd.Wait()
	}()

	// Update build log
	abort := make(chan bool, 1)
	finishBuildLog := make(chan bool)
	go updateBuildLog(config, build.Id, build_log.Name(), abort, finishBuildLog)

	if build.Timeout <= 0 {
		build.Timeout = DEFAULT_TIMEOUT
	}

	var buildState BuildState
	var buildMessage string
	// Wait for signals: abort, timeout or finish
	log.Debugln(config.ShortDescription(), build.Id, "Waiting for signals...")
	select {
	case <-abort:
		log.Println(config.ShortDescription(), build.Id, "Build got aborted.")
		buildState = Failed

	case <-time.After(time.Duration(build.Timeout) * time.Second):
		log.Println(config.ShortDescription(), build.Id, "Build timedout.")
		// command timeout
		if err := cmd.Process.Kill(); err != nil {
		}
		buildState = Failed
		buildMessage = fmt.Sprintf("\nCI Timeout. Execution took longer then %d seconds", build.Timeout)

	case err := <-command_finish:
		// command finished
		if err != nil {
			log.Println(config.ShortDescription(), build.Id, "Build failed with", err)
			buildState = Failed
			buildMessage = fmt.Sprintf("\nBuild failed with %s", err.Error())
		} else {
			log.Println(config.ShortDescription(), build.Id, "Build succeeded.")
			buildState = Success
		}
	}

	// wait for update log routine to finish
	log.Debugln(config.ShortDescription(), build.Id, "Waiting for build log updater to finish")
	finishBuildLog <- true
	log.Debugln(config.ShortDescription(), build.Id, "Build log updater finished.")

	// Send final build state to server
	go func() {
		for {
			if sendBuildLog(config, build.Id, build_log.Name(), buildState, buildMessage) != UpdateFailed {
				break
			} else {
				time.Sleep(UPDATE_RETRY_INTERVAL * time.Second)
			}
		}

		log.Println(config.ShortDescription(), build.Id, "Build finished.")
	}()
	return nil
}
