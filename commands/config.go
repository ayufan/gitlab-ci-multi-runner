package commands

import (
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"os"
)

type configOptions struct {
	config *common.Config

	ConfigFile string `short:"c" long:"config" env:"CONFIG_FILE" description:"Config file"`
}

func (c *configOptions) saveConfig() error {
	return c.config.SaveConfig(c.ConfigFile)
}

func (c *configOptions) loadConfig() error {
	config := common.NewConfig()
	err := config.LoadConfig(c.ConfigFile)
	if err != nil {
		return err
	}
	c.config = config
	return nil
}

func (c *configOptions) touchConfig() error {
	// try to load existing config
	err := c.loadConfig()
	if err != nil {
		return err
	}

	// save config for the first time
	if !c.config.Loaded {
		return c.saveConfig()
	}
	return nil
}

func init() {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		os.Setenv("CONFIG_FILE", getDefaultConfigFile())
	}
}
