package src

import (
	"github.com/codegangsta/cli"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
)

func runSingle(c *cli.Context) {
	runner := RunnerConfig{
		URL:   c.String("URL"),
		Token: c.String("token"),
	}

	log.Println("Starting runner for", runner.URL, "with token", runner.ShortDescription(), "...")

	for {
		build_data, healthy := GetBuild(runner)
		if !healthy {
			log.Println("Runner died, beacuse it's not healthy!")
			os.Exit(1)
		}
		if build_data == nil {
			time.Sleep(CHECK_INTERVAL * time.Second)
			continue
		}

		new_build := Build{
			GetBuildResponse: *build_data,
			Runner:           &runner,
		}
		new_build.Run()
	}
}
