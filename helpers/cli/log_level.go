package cli_helpers

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"os"
	"os/signal"
	"runtime"
	"syscall"
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

			// On USR1 dump stacks of all go routines
			dumpStacks := make(chan os.Signal, 1)
			signal.Notify(dumpStacks, syscall.SIGUSR1)
			go func() {
				for _ = range dumpStacks {
					buf := make([]byte, 1<<20)
					runtime.Stack(buf, true)
					log.Printf("=== received SIGUSR1 ===\n*** goroutine dump...\n%s\n*** end\n", buf)
				}
			}()
		}

		if appBefore != nil {
			return appBefore(c)
		}
		return nil
	}
}
