package commands

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
	"os"
)

type ArtifactCommand struct {
	Archive string `long:"archive" description:"The archive containing your build artifacts"`
	Config  string `long:"config" description:"The client's configuration"`
	Build   int    `long:"build" description:"The build ID to upload artifacts for"`
	Silent  bool   `long:"silent" description:"Suppress output"`
}

func (c *ArtifactCommand) Execute(context *cli.Context) {
	if len(c.Archive) == 0 {
		logrus.Fatalln("Missing archive file")
	}
	if len(c.Config) == 0 {
		logrus.Fatalln("Missing client config")
	}
	if c.Build <= 0 {
		logrus.Fatalln("Missing build ID")
	}

	configData, err := base64.StdEncoding.DecodeString(c.Config)
	if err != nil {
		logrus.Fatalln("Client config in bad format")
	}

	configBuf := bytes.NewReader(configData)

	var config common.RunnerConfig

	if err := json.NewDecoder(configBuf).Decode(&config); err != nil {
		logrus.Fatalln("Client config could not be parsed")
	}

	file, err := os.OpenFile(c.Archive, os.O_RDONLY, os.ModePerm)
	if os.IsNotExist(err) {
		logrus.Fatalln("Archive file did not exist")
	}
	if err != nil {
		logrus.Fatalln("Error opening archive file:", err)
	}
	defer file.Close()

	gl := network.GitLabClient{}

	// If the upload fails, exit with a non-zero exit code to indicate an issue?
	if !gl.UploadArtifacts(config, c.Build, file) {
		os.Exit(1)
	}
}

func init() {
	common.RegisterCommand2("artifacts", "upload build artifacts (internal)", &ArtifactCommand{})
}
