package commands_helpers

import (
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type ExtractCommand struct {
	File    string `long:"file" description:"The file to extract"`
	Verbose bool   `long:"verbose" description:"Suppress archiving output"`

	wd      string
}

func (c *ExtractCommand) extract() {
	logrus.Infoln("Extracting archive", filepath.Base(c.File), "...")

	archive, err := thargo.NewArchiveFile(c.File, nil)
	if err != nil {
		logrus.Fatalln("Failed to open archive", err)
	}

	err = archive.Extract(func(entry thargo.SaveableEntry) error {
		if !c.Verbose {
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

	if c.File == "" {
		logrus.Fatalln("Missing archive file name!")
	}

	c.extract()
}

func init() {
	common.RegisterCommand2("extract", "extract files from an archive (internal)", &ExtractCommand{})
}
