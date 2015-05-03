package commands

import (
	"github.com/codegangsta/cli"

	log "github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

func runUnregister(c *cli.Context) {
	runner := common.RunnerConfig{
		URL:   c.String("url"),
		Token: c.String("token"),
	}

	if !common.DeleteRunner(runner.URL, runner.Token) {
		log.Fatalln("Failed to delete runner")
	}

	config := common.NewConfig()
	err := config.LoadConfig(c.String("config"))
	if err != nil {
		return
	}

	runners := []*common.RunnerConfig{}
	for _, otherRunner := range config.Runners {
		if otherRunner.Token == runner.Token && otherRunner.URL == runner.URL {
			continue
		}
		runners = append(runners, otherRunner)
	}

	// check if anything changed
	if len(config.Runners) == len(runners) {
		return
	}

	config.Runners = runners

	// save config file
	err = config.SaveConfig(c.String("config"))
	if err != nil {
		log.Fatalln("Failed to update", c.String("config"), err)
	}
	log.Println("Updated", c.String("config"))
}

func init() {
	common.RegisterCommand(cli.Command{
		Name:   "unregister",
		Usage:  "unregister specific runner",
		Action: runUnregister,
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
			cli.StringFlag{
				Name:   "c, config",
				Value:  "config.toml",
				Usage:  "Config file",
				EnvVar: "CONFIG_FILE",
			},
		},
	})
}
