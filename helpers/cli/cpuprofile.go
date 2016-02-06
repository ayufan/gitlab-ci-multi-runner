package cli_helpers

import (
	"os"
	"runtime/pprof"

	"github.com/codegangsta/cli"
)

func SetupCpuProfile(app *cli.App) {
	app.Flags = append(app.Flags, cli.StringFlag{
		Name:   "cpuprofile",
		Usage:  "write cpu profile to file",
		EnvVar: "CPU_PROFILE",
	})

	appBefore := app.Before
	appAfter := app.After

	app.Before = func(c *cli.Context) error {
		if cpuProfile := c.String("cpuprofile"); cpuProfile != "" {
			f, err := os.Create(cpuProfile)
			if err != nil {
				return err
			}
			pprof.StartCPUProfile(f)
		}

		if appBefore != nil {
			return appBefore(c)
		} else {
			return nil
		}
	}

	app.After = func(c *cli.Context) error {
		pprof.StopCPUProfile()

		if appAfter != nil {
			return appAfter(c)
		} else {
			return nil
		}
	}
}
