package commands_helpers

import (
	"io"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/archives"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/formatter"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
)

type ArtifactsUploaderCommand struct {
	common.BuildCredentials
	FileArchiver
}

func (c *ArtifactsUploaderCommand) createAndUpload(network common.Network) common.UploadState {
	pr, pw := io.Pipe()
	defer pr.Close()

	// Create the archive
	go func() {
		err := archives.CreateZipArchive(pw, c.sortedFiles())
		pw.CloseWithError(err)
	}()

	// Upload the data
	return network.UploadRawArtifacts(c.BuildCredentials, pr, "artifacts.zip")
}

func (c *ArtifactsUploaderCommand) Execute(*cli.Context) {
	formatter.SetRunnerFormatter()

	if len(c.URL) == 0 || len(c.Token) == 0 {
		logrus.Fatalln("Missing runner credentials")
	}
	if c.ID <= 0 {
		logrus.Fatalln("Missing build ID")
	}

	// Enumerate files
	err := c.enumerate()
	if err != nil {
		logrus.Fatalln(err)
	}

	gl := network.GitLabClient{}

	// If the upload fails, exit with a non-zero exit code to indicate an issue?
retry:
	for i := 0; i < 3; i++ {
		switch c.createAndUpload(&gl) {
		case common.UploadSucceeded:
			return
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
	common.RegisterCommand2("artifacts-uploader", "create and upload build artifacts (internal)", &ArtifactsUploaderCommand{})
}
