package main

import (
	"time"
	"github.com/codegangsta/cli"
)

func run(c *cli.Context) {
	runner_config := RunnerConfig{
		URL: flURL.Value,
		Token: flToken.Value,
	}

	for {
		new_build := GetBuild(&runner_config)
		if new_build != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		build := Build{new_build}
		build.Run()
	}
}
