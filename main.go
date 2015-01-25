package main

import (
	"os"
	"path"
	"fmt"

	"github.com/codegangsta/cli"
	log "github.com/Sirupsen/logrus"
)

func main() {
	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "a GitLab-CI Multi Runner"
	app.Version = "0.1.0"
	app.Author = ""
	app.Email = ""

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "debug mode",
			EnvVar: "DEBUG",
		},
		cli.StringFlag{
			Name:  "log-level, l",
			Value: "info",
			Usage: fmt.Sprintf("Log level (options: debug, info, warn, error, fatal, panic)"),
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

	app.Commands = []cli.Command{
		{
			Name:      "run",
			ShortName: "r",
			Usage:     "start single runner",
			Flags:     []cli.Flag{flToken, flURL},
			Action:    run,
		},
		{
			Name:      "setup",
			ShortName: "s",
			Usage:     "setup a new runner",
			Flags:     []cli.Flag{flRegistrationToken, flURL, flHostname},
			Action:    setup,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
