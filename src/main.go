package src

import (
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

func Main() {
	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "a GitLab-CI Multi Runner"
	app.Version = "0.1.0"
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

	app.Commands = []cli.Command{
		{
			Name:      "run-single",
			ShortName: "rs",
			Usage:     "start single runner",
			Flags:     []cli.Flag{flToken, flURL},
			Action:    runSingle,
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
