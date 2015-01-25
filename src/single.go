package src

import (
	"bytes"
	"errors"
	"github.com/codegangsta/cli"
	"time"

	log "github.com/Sirupsen/logrus"
)

func failBuild(config RunnerConfig, build Build, err error) {
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
			time.Sleep(3 * time.Second)
			continue
		}
	}
}

func runSingle(c *cli.Context) {
	runner_config := RunnerConfig{
		URL:   c.String("URL"),
		Token: c.String("token"),
	}

	println("Starting runner for", runner_config.URL, "with token", runner_config.Token, "...")

	for {
		new_build := GetBuild(runner_config)
		if new_build == nil {
			time.Sleep(3 * time.Second)
			continue
		}

		build := Build{*new_build}

		executor := GetExecutor(runner_config)
		if executor == nil {
			go failBuild(runner_config, build, errors.New("couldn't get executor"))
			continue
		}

		err := executor.Run(runner_config, build)
		if err != nil {
			go failBuild(runner_config, build, err)
			continue
		}
	}
}
