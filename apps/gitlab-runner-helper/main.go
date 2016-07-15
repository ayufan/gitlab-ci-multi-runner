package main

import (
	"os"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/formatter"

	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/commands/helpers"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			// log panics forces exit
			if _, ok := r.(*logrus.Entry); ok {
				os.Exit(1)
			}
			panic(r)
		}
	}()

	formatter.SetRunnerFormatter()

	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "a GitLab Runner Helper"
	app.Version = common.AppVersion.ShortLine()
	cli.VersionPrinter = common.AppVersion.Printer
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Kamil Trzciński",
			Email: "ayufan@ayufan.eu",
		},
	}
	cli_helpers.SetupLogLevelOptions(app)
	app.Commands = common.GetCommands()
	app.CommandNotFound = func(context *cli.Context, command string) {
		logrus.Fatalln("Command", command, "not found")
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
