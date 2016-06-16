package main

import (
	"os"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/formatter"

	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/commands"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/commands/helpers"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/docker"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/docker/machine"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/parallels"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/shell"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/ssh"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/virtualbox"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/shells"
)

var NAME = "gitlab-ci-multi-runner"
var VERSION = "dev"
var REVISION = "HEAD"
var BUILT = "now"
var BRANCH = "HEAD"

func init() {
	common.NAME = NAME
	common.VERSION = VERSION
	common.REVISION = REVISION
	common.BUILT = BUILT
	common.BRANCH = BRANCH
}

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

	// Start background reaping of orphaned child processes.
	// It allows the gitlab-runner to act as `init` process
	go helpers.Reap()

	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "a GitLab Runner"
	app.Version = common.VersionShortLine()
	cli.VersionPrinter = common.VersionPrinter
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Kamil Trzciński",
			Email: "ayufan@ayufan.eu",
		},
	}
	cli_helpers.LogRuntimePlatform(app)
	cli_helpers.SetupLogLevelOptions(app)
	cli_helpers.SetupCPUProfile(app)
	cli_helpers.FixHOME(app)
	app.Commands = common.GetCommands()
	app.CommandNotFound = func(context *cli.Context, command string) {
		logrus.Fatalln("Command", command, "not found.")
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
