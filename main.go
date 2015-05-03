package main

import (
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"fmt"
	_ "github.com/ayufan/gitlab-ci-multi-runner/commands"
	"github.com/ayufan/gitlab-ci-multi-runner/common"
	_ "github.com/ayufan/gitlab-ci-multi-runner/executors/docker"
	_ "github.com/ayufan/gitlab-ci-multi-runner/executors/parallels"
	_ "github.com/ayufan/gitlab-ci-multi-runner/executors/shell"
	_ "github.com/ayufan/gitlab-ci-multi-runner/executors/ssh"
	_ "github.com/ayufan/gitlab-ci-multi-runner/shells"
)

func main() {
	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "a GitLab Runner"
	app.Version = fmt.Sprintf("%s (%s)", common.VERSION, common.REVISION)
	app.Author = "Kamil Trzciński"
	app.Email = "ayufan@ayufan.eu"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "debug mode",
			EnvVar: "DEBUG",
		},
		cli.StringFlag{
			Name:  "log-level, l",
			Value: "info",
			Usage: "Log level (options: debug, info, warn, error, fatal, panic)",
		},
	}

	// logs
	app.Before = func(c *cli.Context) error {
		log.SetOutput(os.Stderr)
		level, err := log.ParseLevel(c.String("log-level"))
		if err != nil {
			log.Fatalf(err.Error())
		}
		log.SetLevel(level)

		// If a log level wasn't specified and we are running in debug mode,
		// enforce log-level=debug.
		if !c.IsSet("log-level") && !c.IsSet("l") && c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	}

	app.Commands = common.GetCommands()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
