package commands_helpers

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/archives"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/formatter"
)

type CacheExtractorCommand struct {
	File string `long:"file" description:"The file containing your cache artifacts"`
}

func (c *CacheExtractorCommand) Execute(context *cli.Context) {
	formatter.SetRunnerFormatter()

	if len(c.File) == 0 {
		logrus.Fatalln("Missing cache file")
	}

	err := archives.ExtractZipFile(c.File)
	if err != nil && !os.IsNotExist(err) {
		logrus.Fatalln(err)
	}
}

func init() {
	common.RegisterCommand2("cache-extractor", "download and extract cache artifacts (internal)", &CacheExtractorCommand{})
}
