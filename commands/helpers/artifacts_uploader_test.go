package commands_helpers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io/ioutil"
)

var UploaderCredentials = common.BuildCredentials{
	ID:    1000,
	Token: "test",
	URL:   "test",
}

func TestArtifactsUploaderRequirements(t *testing.T) {
	helpers.MakeFatalToPanic()

	cmd := ArtifactsUploaderCommand{}
	assert.Panics(t, func() {
		cmd.Execute(nil)
	})
}

func TestArtifactsUploaderTooLarge(t *testing.T) {
	network := &testNetwork{
		uploadState: common.UploadTooLarge,
	}
	cmd := ArtifactsUploaderCommand{
		BuildCredentials: UploaderCredentials,
		network:          network,
	}

	assert.Panics(t, func() {
		cmd.Execute(nil)
	})

	assert.Equal(t, 1, network.uploadCalled)
}

func TestArtifactsUploaderForbidden(t *testing.T) {
	network := &testNetwork{
		uploadState: common.UploadForbidden,
	}
	cmd := ArtifactsUploaderCommand{
		BuildCredentials: UploaderCredentials,
		network:          network,
	}

	assert.Panics(t, func() {
		cmd.Execute(nil)
	})

	assert.Equal(t, 1, network.uploadCalled)
}

func TestArtifactsUploaderRetry(t *testing.T) {
	network := &testNetwork{
		uploadState: common.UploadFailed,
	}
	cmd := ArtifactsUploaderCommand{
		BuildCredentials: UploaderCredentials,
		network:          network,
		retryHelper: retryHelper{
			Retry: 2,
		},
	}

	assert.Panics(t, func() {
		cmd.Execute(nil)
	})

	assert.Equal(t, 3, network.uploadCalled)
}

func TestArtifactsUploaderSucceeded(t *testing.T) {
	network := &testNetwork{
		uploadState: common.UploadSucceeded,
	}
	cmd := ArtifactsUploaderCommand{
		BuildCredentials: UploaderCredentials,
		network:          network,
		fileArchiver: fileArchiver{
			Paths: []string{artifactsTestArchivedFile},
		},
	}

	ioutil.WriteFile(artifactsTestArchivedFile, nil, 0600)
	defer os.Remove(artifactsTestArchivedFile)

	cmd.Execute(nil)
	assert.Equal(t, 1, network.uploadCalled)
	fi, _ := os.Stat(artifactsTestArchivedFile)
	assert.NotNil(t, fi)
}
