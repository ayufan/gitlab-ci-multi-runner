package commands

import (
	"github.com/codegangsta/cli"

	log "github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

func runVerify(c *cli.Context) {
	config := common.NewConfig()
	err := config.LoadConfig(c.String("config"))
	if err != nil {
		log.Fatalln(err)
		return
	}

	// verify if runner exist
	runners := []*common.RunnerConfig{}
	for _, runner := range config.Runners {
		if common.VerifyRunner(runner.URL, runner.Token) {
			runners = append(runners, runner)
		}
	}

	if !c.Bool("delete") {
		return
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
		Name:   "verify",
		Usage:  "verify all registered runners",
		Action: runVerify,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "c, config",
				Value:  getDefaultConfigFile(),
				Usage:  "Config file",
				EnvVar: "CONFIG_FILE",
			},
			cli.BoolFlag{
				Name:  "delete",
				Usage: "Delete no longer existing runners?",
			},
		},
	})
}
