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
		EnvVar: "CI_SERVER_URL",
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
	flDescription = cli.StringFlag{
		Name:   "description",
		Value:  "",
		Usage:  "Runner's registration description",
		EnvVar: "RUNNER_DESCRIPTION",
	}
	flTags = cli.StringFlag{
		Name:   "tag-list",
		Value:  "",
		Usage:  "Runner's tag list separated by comma",
		EnvVar: "RUNNER_TAG_LIST",
	}
	flExecutor = cli.StringFlag{
		Name:   "executor",
		Value:  "",
		Usage:  "Select executor, eg. shell, docker, etc.",
		EnvVar: "RUNNER_EXECUTOR",
	}
	flDockerHost = cli.StringFlag{
		Name:   "docker-host",
		Value:  "",
		Usage:  "Docker endpoint URL",
		EnvVar: "DOCKER_HOST",
	}
)
