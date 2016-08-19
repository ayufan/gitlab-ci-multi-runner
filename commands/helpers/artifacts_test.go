package helpers

import (
	"archive/zip"
	"bytes"
	"io"
	"os"

	"github.com/Sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

const artifactsTestArchivedFile = "archive_file"

type testNetwork struct {
	common.MockNetwork
	downloadState  common.DownloadState
	downloadCalled int
	uploadState    common.UploadState
	uploadCalled   int
}

func (m *testNetwork) DownloadArtifacts(config common.BuildCredentials, artifactsFile string) common.DownloadState {
	m.downloadCalled++

	if m.downloadState == common.DownloadSucceeded {
		file, err := os.Create(artifactsFile)
		if err != nil {
			logrus.Warningln(err)
			return common.DownloadFailed
		}
		defer file.Close()

		archive := zip.NewWriter(file)
		archive.Create(artifactsTestArchivedFile)
		archive.Close()
	}
	return m.downloadState
}

func (m *testNetwork) UploadRawArtifacts(config common.BuildCredentials, reader io.Reader, baseName string, expireIn string) common.UploadState {
	m.uploadCalled++

	if m.uploadState == common.UploadSucceeded {
		var buffer bytes.Buffer
		io.Copy(&buffer, reader)
		archive, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
		if err != nil {
			logrus.Warningln(err)
			return common.UploadForbidden
		}

		if len(archive.File) != 1 || archive.File[0].Name != artifactsTestArchivedFile {
			logrus.Warningln("Invalid archive:", len(archive.File))
			return common.UploadForbidden
		}
	}
	return m.uploadState
}
