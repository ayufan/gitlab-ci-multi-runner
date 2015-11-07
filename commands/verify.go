package commands

import (
	"github.com/codegangsta/cli"

	log "github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
)

type VerifyCommand struct {
	configOptions
	network common.Network

	DeleteNonExisting bool `long:"delete" description:"Delete no longer existing runners?"`
}

func (c *VerifyCommand) Execute(context *cli.Context) {
	err := c.loadConfig()
	if err != nil {
		log.Fatalln(err)
		return
	}

	// verify if runner exist
	runners := []*common.RunnerConfig{}
	for _, runner := range c.config.Runners {
		if c.network.VerifyRunner(runner.RunnerCredentials) {
			runners = append(runners, runner)
		}
	}

	if !c.DeleteNonExisting {
		return
	}

	// check if anything changed
	if len(c.config.Runners) == len(runners) {
		return
	}

	c.config.Runners = runners

	// save config file
	err = c.saveConfig()
	if err != nil {
		log.Fatalln("Failed to update", c.ConfigFile, err)
	}
	log.Println("Updated", c.ConfigFile)
}

func init() {
	common.RegisterCommand2("verify", "verify all registered runners", &VerifyCommand{
		network: &network.GitLabClient{},
	})
}
