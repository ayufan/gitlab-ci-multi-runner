package cli_helpers

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/docker/docker/pkg/homedir"
	"github.com/Sirupsen/logrus"
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
