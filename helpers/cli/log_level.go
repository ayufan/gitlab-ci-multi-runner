package cli_helpers

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"os"
)

func SetupLogLevelOptions(app *cli.App) {
	newFlags := []cli.Flag{
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
	app.Flags = append(app.Flags, newFlags...)

	appBefore := app.Before
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
			go watchForGoroutinesDump()
		}

		if appBefore != nil {
			return appBefore(c)
		}
		return nil
	}
}
