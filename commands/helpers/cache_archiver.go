package commands_helpers

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/archives"
)

type CacheArchiverCommand struct {
	fileArchiver
	File string `long:"file" description:"The path to file"`
}

func (c *CacheArchiverCommand) Execute(*cli.Context) {
	if c.File == "" {
		logrus.Fatalln("Missing --file")
	}

	// Enumerate files
	err := c.enumerate()
	if err != nil {
		logrus.Fatalln(err)
	}

	// Check if list of files changed
	if !c.isFileChanged(c.File) {
		logrus.Infoln("Archive is up to date!")
		return
	}

	// Create archive
	err = archives.CreateZipFile(c.File, c.sortedFiles())
	if err != nil {
		logrus.Fatalln(err)
	}
}

func init() {
	common.RegisterCommand2("cache-archiver", "create and upload cache artifacts (internal)", &CacheArchiverCommand{})
}
