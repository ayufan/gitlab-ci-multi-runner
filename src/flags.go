package src

import (
	"github.com/codegangsta/cli"
)

var (
	flConfigFile = cli.StringFlag{
		Name:   "config",
		Value:  "config.toml",
		Usage:  "Config file",
		EnvVar: "CONFIG_FILE",
	}
	flURL = cli.StringFlag{
		Name:   "url",
		Value:  "",
		Usage:  "Runner URL",
		EnvVar: "RUNNER_URL",
	}
	flToken = cli.StringFlag{
		Name:   "token",
		Value:  "",
		Usage:  "Runner token",
		EnvVar: "RUNNER_TOKEN",
	}
	flRegistrationToken = cli.StringFlag{
		Name:   "registration-token",
		Value:  "",
		Usage:  "Runner's registration token",
		EnvVar: "REGISTRATION_TOKEN",
	}
	flHostname = cli.StringFlag{
		Name:   "hostname",
		Value:  "",
		Usage:  "Runner's registration hostname",
		EnvVar: "HOSTNAME",
	}
)
