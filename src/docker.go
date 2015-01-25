package src

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
)

type DockerExecutor struct {
}

func (s *DockerExecutor) Run(config RunnerConfig, build Build) error {
	builds_dir := "/builds"
	if len(config.BuildsDir) != 0 {
		builds_dir = config.BuildsDir
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

	// connect to docker
	endpoint := "unix:///var/run/docker.sock"
	client, _ := docker.NewClient(endpoint)
	err = client.Ping()
	if err != nil {
		return err
	}

	create_container_opts := docker.CreateContainerOptions{
		Name: build.ProjectUniqueName(),
		Config: &docker.Config{
			Hostname:    build.ProjectUniqueName(),
			Image:       config.DockerImage,
			AttachStdin: true,
			Env: []string{
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
			},
			Cmd: []string{"bash", "-"},
		},
		HostConfig: &docker.HostConfig{
			Privileged: config.DockerPrivileged,
		},
	}

	if !config.DockerDisableCache && build.AllowGitFetch {
		// create temporary volume

	}

	container, err := client.CreateContainer(create_container_opts)
	if err != nil {
		return err
	}
	remove_container_opts := docker.RemoveContainerOptions{
		ID:            container.ID,
		RemoveVolumes: true,
		Force:         true,
	}
	defer client.RemoveContainer(remove_container_opts)

	attach_container_opts := docker.AttachToContainerOptions{
		Container:    container.ID,
		InputStream:  nil,
		OutputStream: build_log,
		ErrorStream:  build_log,
		Logs:         true,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
	}

	err = client.AttachToContainer(attach_container_opts)
	if err != nil {
		return err
	}

	// Update build log
	abort := make(chan bool, 1)
	finishBuildLog := make(chan bool)
	go build.WatchTrace(config, abort, finishBuildLog)

	// Wait for process to exit
	command_finish := make(chan int, 1)
	go func() {
		exit_code, _ := client.WaitContainer(container.ID)
		command_finish <- exit_code
	}()

	if build.Timeout <= 0 {
		build.Timeout = DEFAULT_TIMEOUT
	}

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
		buildState = Failed
		buildMessage = fmt.Sprintf("\nCI Timeout. Execution took longer then %d seconds", build.Timeout)

	case exit_code := <-command_finish:
		// command finished
		if exit_code != 0 {
			log.Println(config.ShortDescription(), build.Id, "Build failed with", exit_code)
			buildState = Failed
			buildMessage = fmt.Sprintf("\nBuild failed with %d", exit_code)
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
