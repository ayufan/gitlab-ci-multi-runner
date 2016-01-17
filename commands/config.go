package commands

import (
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
	"os"
	"path/filepath"
)

func getDefaultConfigFile() string {
	return filepath.Join(getDefaultConfigDirectory(), "config.toml")
}

func getDefaultCertificateDirectory() string {
	return filepath.Join(getDefaultConfigDirectory(), "certs")
}

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

func (c *configOptions) RunnerByName(name string) (*common.RunnerConfig, error) {
	if c.config == nil {
		return nil, fmt.Errorf("Config has not been loaded")
	}

	for _, runner := range c.config.Runners {
		if runner.Name == name {
			return runner, nil
		}
	}

	return nil, fmt.Errorf("Could not find a runner with the name '%s'", name)
}

func init() {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		os.Setenv("CONFIG_FILE", getDefaultConfigFile())
	}

	network.CertificateDirectory = getDefaultCertificateDirectory()
}
