package commands_helpers

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
	"os"
	"time"
)

type ArtifactCommand struct {
	common.RunnerCredentials
	Archive string `long:"archive" description:"The archive containing your build artifacts"`
	Build   int    `long:"build-id" description:"The build ID to upload artifacts for"`
	Silent  bool   `long:"silent" description:"Suppress output"`
}

func (c *ArtifactCommand) Execute(context *cli.Context) {
	if len(c.Archive) == 0 {
		logrus.Fatalln("Missing archive file")
	}
	if len(c.URL) == 0 || len(c.Token) == 0 {
		logrus.Fatalln("Missing runner credentials")
	}
	if c.Build <= 0 {
		logrus.Fatalln("Missing build ID")
	}

	gl := network.GitLabClient{}

	// If the upload fails, exit with a non-zero exit code to indicate an issue?
retry:
	for i := 0; i < 3; i++ {
		switch gl.UploadArtifacts(c.RunnerCredentials, c.Build, c.Archive) {
		case common.UploadSucceeded:
			os.Exit(0)
		case common.UploadForbidden:
			break retry
		case common.UploadTooLarge:
			break retry
		case common.UploadFailed:
			// wait one second to retry
			logrus.Warningln("Retrying...")
			time.Sleep(time.Second)
			break
		}
	}

	os.Exit(1)
}

func init() {
	common.RegisterCommand2("artifacts", "upload build artifacts (internal)", &ArtifactCommand{})
}
