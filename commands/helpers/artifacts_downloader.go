package commands_helpers

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/archives"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/formatter"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
)

type ArtifactsDownloaderCommand struct {
	common.BuildCredentials
}

func (c *ArtifactsDownloaderCommand) download(file string) error {
	gl := network.GitLabClient{}

	// If the download fails, exit with a non-zero exit code to indicate an issue?
retry:
	for i := 0; i < 3; i++ {
		switch gl.DownloadArtifacts(c.BuildCredentials, file) {
		case common.DownloadSucceeded:
			return nil
		case common.DownloadNotFound:
			return os.ErrNotExist
		case common.DownloadForbidden:
			break retry
		case common.DownloadFailed:
			// wait one second to retry
			logrus.Warningln("Retrying...")
			time.Sleep(time.Second)
			break
		}
	}
	return errors.New("Failed to download artifacts")
}

func (c *ArtifactsDownloaderCommand) Execute(context *cli.Context) {
	formatter.SetRunnerFormatter()

	if len(c.URL) == 0 || len(c.Token) == 0 {
		logrus.Fatalln("Missing runner credentials")
	}
	if c.ID <= 0 {
		logrus.Fatalln("Missing build ID")
	}

	// Create temporary file
	file, err := ioutil.TempFile("", "artifacts")
	if err != nil {
		logrus.Fatalln(err)
	}
	file.Close()
	defer os.Remove(file.Name())

	// Download artifacts file
	err = c.download(file.Name())
	if err != nil {
		logrus.Fatalln(err)
	}

	// Extract artifacts file
	err = archives.ExtractZipFile(file.Name())
	if err != nil {
		logrus.Fatalln(err)
	}
}

func init() {
	common.RegisterCommand2("artifacts-downloader", "download and extract build artifacts (internal)", &ArtifactsDownloaderCommand{})
}
