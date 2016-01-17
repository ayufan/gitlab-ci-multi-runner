package commands

import (
	"github.com/codegangsta/cli"

	log "github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
)

type UnregisterCommand struct {
	configOptions
	common.RunnerCredentials
	network common.Network
}

func (c *UnregisterCommand) Execute(context *cli.Context) {
	userModeWarning(false)

	if !c.network.DeleteRunner(c.RunnerCredentials) {
		log.Fatalln("Failed to delete runner")
	}

	err := c.loadConfig()
	if err != nil {
		log.Warningln(err)
		return
	}

	runners := []*common.RunnerConfig{}
	for _, otherRunner := range c.config.Runners {
		if otherRunner.RunnerCredentials == c.RunnerCredentials {
			continue
		}
		runners = append(runners, otherRunner)
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
	common.RegisterCommand2("unregister", "unregister specific runner", &UnregisterCommand{
		network: &network.GitLabClient{},
	})
}
