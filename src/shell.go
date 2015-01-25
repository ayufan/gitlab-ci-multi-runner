package src

import (
	"errors"
	"os"
	"time"
	"fmt"
	"bytes"
	"io/ioutil"
	"os/exec"
)

type ShellExecutor struct {

}

func sendBuildLog(config RunnerConfig, buildId int, build_log string, state BuildState) UpdateState {
	file, err := os.Open(build_log)
	if err != nil {
		return UpdateBuild(config, buildId, state, bytes.NewBufferString(""))
	}
	defer file.Close()

	return UpdateBuild(config, buildId, state, file)
}

func updateBuildLog(config RunnerConfig, buildId int, build_log string, abort chan bool, finished chan bool) {
	for {
		select {
		case <-time.After(time.Second * 3):
			switch sendBuildLog(config, buildId, build_log, Running) {
			case UpdateSucceeded:
			case UpdateAbort:
				abort <- true
				<- finished
				return
			case UpdateFailed:
			}

		case <-finished:
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

	// create build log
	build_log, err := ioutil.TempFile("", "build_log")
	if err != nil {
		return errors.New("Failed to create build log file")
	}
	defer build_log.Close()
	defer os.Remove(build_log.Name())

	// create execution command
	cmd := exec.Command("setsid", *script_file)
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

	// Wait for process to exit
	command_finish := make(chan error, 1)
	go func() {
		command_finish <- cmd.Wait()
	}()

	// Update build log
	abort := make(chan bool, 1)
	finishBuildLog := make(chan bool)
	go updateBuildLog(config, build.Id, build_log.Name(), abort, finishBuildLog)

	var buildState BuildState

	// Wait for signals: abort, timeout or finish
	select {
	case <-abort:
		// abort build
		buildState = Failed
	case <-time.After(time.Second * time.Duration(build.Timeout)):
		// command timeout
		if err := cmd.Process.Kill(); err != nil {
		}
		buildState = Failed
	case err := <-command_finish:
		// command finished
		if err != nil {
			buildState = Failed
		} else {
			buildState = Success
		}
	}

	// wait for update log routine to finish
	finishBuildLog <- true

	// Send final build state to server
	for {
		switch sendBuildLog(config, build.Id, build_log.Name(), buildState) {
		case UpdateSucceeded:
			return nil
		case UpdateAbort:
			return nil
		case UpdateFailed:
			time.Sleep(3 * time.Second)
		}
	}

	return nil
}
