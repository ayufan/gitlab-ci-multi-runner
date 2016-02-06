package commands_helpers

import (
	"testing"

	"archive/zip"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io/ioutil"
	"os"
)

const cacheExtractorArchive = "archive.zip"
const cacheExtractorTestArchivedFile = "archive_file"

func TestCacheExtractorValidArchive(t *testing.T) {
	file, err := os.Create(cacheExtractorArchive)
	assert.NoError(t, err)
	defer file.Close()
	defer os.Remove(file.Name())
	defer os.Remove(cacheExtractorTestArchivedFile)

	archive := zip.NewWriter(file)
	archive.Create(cacheExtractorTestArchivedFile)
	archive.Close()

	_, err = os.Stat(cacheExtractorTestArchivedFile)
	assert.Error(t, err)

	cmd := CacheExtractorCommand{
		File: cacheExtractorArchive,
	}
	assert.NotPanics(t, func() {
		cmd.Execute(nil)
	})

	_, err = os.Stat(cacheExtractorTestArchivedFile)
	assert.NoError(t, err)
}

func TestCacheExtractorForInvalidArchive(t *testing.T) {
	helpers.MakeFatalToPanic()
	ioutil.WriteFile(cacheExtractorArchive, nil, 0600)
	defer os.Remove(cacheExtractorArchive)

	cmd := CacheExtractorCommand{
		File: cacheExtractorArchive,
	}
	assert.Panics(t, func() {
		cmd.Execute(nil)
	})
}

func TestCacheExtractorForIfNoFileDefined(t *testing.T) {
	helpers.MakeFatalToPanic()
	cmd := CacheExtractorCommand{}
	assert.Panics(t, func() {
		cmd.Execute(nil)
	})
}

func TestCacheExtractorForNotExistingFile(t *testing.T) {
	helpers.MakeFatalToPanic()
	cmd := CacheExtractorCommand{
		File: "/../../../test.zip",
	}
	assert.NotPanics(t, func() {
		cmd.Execute(nil)
	})
}
