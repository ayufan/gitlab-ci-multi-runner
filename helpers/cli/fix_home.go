package cli_helpers

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/docker/pkg/homedir"
)

func FixHOME(app *cli.App) {
	appBefore := app.Before

	app.Before = func(c *cli.Context) error {
		key := homedir.Key()
		if os.Getenv(key) != "" {
			return
		}

		value := homedir.Get()
		if value == "" {
			logrus.Fatalln("The", key, "is not set")
			return
		}

		os.Setenv(key, value)

		if appBefore != nil {
			return appBefore(c)
		}
		return nil
	}
}
