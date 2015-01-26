package src

import (
	"bytes"
	"errors"
	"time"

	log "github.com/Sirupsen/logrus"
)

type Job struct {
	Build  *Build
	Runner *RunnerConfig
}

func (j *Job) fail(config RunnerConfig, build Build, err error) {
	log.Println(config.ShortDescription(), build.Id, "Build failed", err)
	for {
		error_buffer := bytes.NewBufferString(err.Error())
		result := UpdateBuild(config, build.Id, Failed, error_buffer)
		switch result {
		case UpdateSucceeded:
			return
		case UpdateAbort:
			return
		case UpdateFailed:
			time.Sleep(UPDATE_RETRY_INTERVAL * time.Second)
			continue
		}
	}
}

func (j *Job) Run() error {
	var err error
	executor := GetExecutor(j.Runner.Executor)
	if executor == nil {
		err = errors.New("executor not found")
	}
	if err == nil {
		err = executor.Prepare(j.Runner, j.Build)
	}
	if err == nil {
		err = executor.Start()
	}
	if err == nil {
		err = executor.Wait()
	}
	if err != nil {
		go failBuild(*j.Runner, *j.Build, err)
	}
	if executor != nil {
		executor.Cleanup()
	}
	return err
}
