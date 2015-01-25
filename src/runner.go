package src

import (
	"errors"
	"time"
	"bytes"
	"github.com/codegangsta/cli"
)

func failBuild(config RunnerConfig, build Build, err error) {
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

func run(c *cli.Context) {
	runner_config := RunnerConfig{
		URL: flURL.Value,
		Token: flToken.Value,
	}

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
