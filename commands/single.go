package commands

import (
	"github.com/codegangsta/cli"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/ayufan/gitlab-ci-multi-runner/common"
)

func runSingle(c *cli.Context) {
	runner := common.RunnerConfig{
		URL:   c.String("URL"),
		Token: c.String("token"),
	}

	log.Println("Starting runner for", runner.URL, "with token", runner.ShortDescription(), "...")

	for {
		build_data, healthy := common.GetBuild(runner)
		if !healthy {
			log.Println("Runner died, beacuse it's not healthy!")
			os.Exit(1)
		}
		if build_data == nil {
			time.Sleep(common.CHECK_INTERVAL * time.Second)
			continue
		}

		new_build := common.Build{
			GetBuildResponse: *build_data,
			Runner:           &runner,
		}
		new_build.Run()
	}
}

var (
	CmdRunSingle = cli.Command{
		Name:      "run-single",
		ShortName: "rs",
		Usage:     "start single runner",
		Action:    runSingle,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "token",
				Value:  "",
				Usage:  "Runner token",
				EnvVar: "RUNNER_TOKEN",
			},
			cli.StringFlag{
				Name:   "url",
				Value:  "",
				Usage:  "Runner URL",
				EnvVar: "CI_SERVER_URL",
			},
		},
	}
)
