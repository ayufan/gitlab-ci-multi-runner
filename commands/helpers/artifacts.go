package commands_helpers

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/formatter"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
	"os"
	"time"
)

type ArtifactCommand struct {
	common.BuildCredentials
	File     string `long:"file" description:"The file containing your build artifacts"`
	Download bool   `long:"download" description:"Download artifacts instead of uploading them"`
}

func (c *ArtifactCommand) upload() {
	gl := network.GitLabClient{}

	// If the upload fails, exit with a non-zero exit code to indicate an issue?
retry:
	for i := 0; i < 3; i++ {
		switch gl.UploadArtifacts(c.BuildCredentials, c.File) {
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

func (c *ArtifactCommand) download() {
	gl := network.GitLabClient{}

	// If the download fails, exit with a non-zero exit code to indicate an issue?
retry:
	for i := 0; i < 3; i++ {
		switch gl.DownloadArtifacts(c.BuildCredentials, c.File) {
		case common.DownloadSucceeded:
			os.Exit(0)
		case common.DownloadForbidden:
			break retry
		case common.DownloadFailed:
			// wait one second to retry
			logrus.Warningln("Retrying...")
			time.Sleep(time.Second)
			break
		}
	}

	os.Exit(1)
}

func (c *ArtifactCommand) Execute(context *cli.Context) {
	formatter.SetRunnerFormatter()

	if len(c.File) == 0 {
		logrus.Fatalln("Missing archive file")
	}
	if len(c.URL) == 0 || len(c.Token) == 0 {
		logrus.Fatalln("Missing runner credentials")
	}
	if c.ID <= 0 {
		logrus.Fatalln("Missing build ID")
	}

	if c.Download {
		c.download()
	} else {
		c.upload()
	}
}

func init() {
	common.RegisterCommand2("artifacts", "download or upload build artifacts (internal)", &ArtifactCommand{})
}
