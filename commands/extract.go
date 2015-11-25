package commands

import (
	"os"
	"path/filepath"

	"github.com/EMSSConsulting/Thargo"
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type ExtractCommand struct {
	Input  string `long:"input" description:"The filepath to the input archive"`
	Silent bool   `long:"silent" description:"Suppress archiving output"`

	wd string
}

func (c *ExtractCommand) extract() {
	logrus.Infoln("Extracting archive", filepath.Base(c.Input), "...")

	archive, err := thargo.NewArchiveFile(c.Input, nil)
	if err != nil {
		logrus.Fatalln("Failed to open archive", err)
	}

	err = archive.Extract(func(entry thargo.SaveableEntry) error {
		if !c.Silent {
			header, err := entry.Header()
			if err != nil {
				return err
			}

			logrus.Infoln(" - ", header.Name)
		}

		return entry.Save(c.wd)
	})

	if err != nil {
		logrus.Fatalln("Failed to extract archive", err)
	}

	logrus.Infoln("Done!")
}

func (c *ExtractCommand) Execute(context *cli.Context) {
	logrus.SetFormatter(
		&logrus.TextFormatter{
			ForceColors:      true,
			DisableTimestamp: false,
		},
	)

	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatalln("Failed to get current working directory:", err)
	}
	if c.Input == "" {
		logrus.Fatalln("Missing archive file name!")
	}

	c.wd = wd
	
	c.extract()
}

func init() {
	common.RegisterCommand2("extract", "extract files from an archive (internal)", &ExtractCommand{})
}
