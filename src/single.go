package src

import (
	"bytes"
	"github.com/codegangsta/cli"
	"os"
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
			time.Sleep(UPDATE_RETRY_INTERVAL * time.Second)
			continue
		}
	}
}

func runSingle(c *cli.Context) {
	runner_config := RunnerConfig{
		URL:   c.String("URL"),
		Token: c.String("token"),
	}

	log.Println("Starting runner for", runner_config.URL, "with token", runner_config.ShortDescription(), "...")

	for {
		new_build, healthy := GetBuild(runner_config)
		if !healthy {
			log.Println("Runner died, beacuse it's not healthy!")
			os.Exit(1)
		}
		if new_build == nil {
			time.Sleep(CHECK_INTERVAL * time.Second)
			continue
		}

		new_job := Job{
			Build: &Build{
				GetBuildResponse: *new_build,
				Name:             "",
			},
			Runner: &runner_config,
		}

		new_job.Run()
	}
}
