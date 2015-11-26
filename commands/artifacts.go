package commands

import (
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
	"os"
)

type ArtifactCommand struct {
	Archive string `long:"archive" description:"The archive containing your build artifacts"`
	ID      string `long:"id" description:"The unique ID of the client"`
	Build   int    `long:"build" description:"The build ID to upload artifacts for"`
	Silent  bool   `long:"silent" description:"Suppress output"`
}

func (c *ArtifactCommand) Execute(context *cli.Context) {
	if len(c.Archive) == 0 {
		logrus.Fatalln("Missing archive file")
	}
	if len(c.ID) == 0 {
		logrus.Fatalln("Missing client ID")
	}
	if c.Build <= 0 {
		logrus.Fatalln("Missing build ID")
	}

	config := &configOptions{}
	if err := config.loadConfig(); err != nil {
		logrus.Fatalln("Failed to load config file, please ensure it is in the default location or CONFIG_PATH is set.")
	}

	var runner *common.RunnerConfig
	for _, r := range config.config.Runners {
		if r.UniqueID() == c.ID {
			runner = r
			break
		}
	}

	if runner == nil {
		logrus.Fatalln("Client ID didn't match a known runner")
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
	if !gl.UploadArtifacts(*runner, c.Build, file) {
		os.Exit(1)
	}
}

func init() {
	common.RegisterCommand2("artifacts", "upload build artifacts (internal)", &ArtifactCommand{})
}