package commands_helpers

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
	"io"
	"os"
	"path/filepath"
	"time"
	"github.com/cheggaaa/pb"
)

type ArtifactCommand struct {
	common.BuildCredentials
	File       string `long:"file" description:"The file containing your build artifacts"`
	NoProgress bool   `long:"no-progress" description:"Disable progress bar"`
}

// It's proxy reader, implement io.Reader
type artifactReader struct {
	io.Reader
	bar *pb.ProgressBar
}

func (r *artifactReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.bar.Add(n)
	return
}

func (r *artifactReader) Close() error {
	r.bar.Finish()
	return nil
}

func (c *ArtifactCommand) upload() common.UploadState {
	file, err := os.Open(c.File)
	if err != nil {
		logrus.Warningln("Failed to open file:", c.File, err)
		time.Sleep(time.Second)
		return common.UploadFailed
	}
	defer file.Close()

	var content io.ReadCloser
	if !c.NoProgress {
		fi, err := file.Stat()
		if err != nil {
			logrus.Warningln("Failed to stat file:", c.File, err)
			return common.UploadFailed
		}
		bar := helpers.NewPbForBytes(fi.Size())
		content = &artifactReader{file, bar}
	} else {
		content = file
	}

	gl := network.GitLabClient{}
	return gl.UploadArtifacts(c.BuildCredentials, content, filepath.Base(c.File))
}

func (c *ArtifactCommand) Execute(context *cli.Context) {
	if len(c.File) == 0 {
		logrus.Fatalln("Missing archive file")
	}
	if len(c.URL) == 0 || len(c.Token) == 0 {
		logrus.Fatalln("Missing runner credentials")
	}
	if c.ID <= 0 {
		logrus.Fatalln("Missing build ID")
	}

	// If the upload fails, exit with a non-zero exit code to indicate an issue?
retry:
	for i := 0; i < 3; i++ {
		switch c.upload() {
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
		}
	}
	os.Exit(1)
}

func init() {
	common.RegisterCommand2("artifacts", "upload build artifacts (internal)", &ArtifactCommand{})
}
