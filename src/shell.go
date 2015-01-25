package src

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	log "github.com/Sirupsen/logrus"
)

type ShellExecutor struct {
}

func (s *ShellExecutor) Run(config RunnerConfig, build Build) error {
	builds_dir := "tmp/builds"
	if len(config.BuildsDir) != 0 {
		builds_dir = c.BuildsDir
	}

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
	build.BuildLog = build_log.Name()
	log.Debugln(config.ShortDescription(), build.Id, "Created build log:", build_log.Name())

	shell_script := config.ShellScript
	if len(shell_script) == 0 {
		shell_script = "setsid"
	}

	// create execution command
	cmd := exec.Command(shell_script, *script_file)
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
	go build.WatchTrace(config, abort, finishBuildLog)

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
	go build.FinishBuild(config, buildState, buildMessage)
	return nil
}
