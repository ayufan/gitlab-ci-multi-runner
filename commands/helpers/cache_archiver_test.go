package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io/ioutil"
	"os"
	"time"
)

const cacheArchiverArchive = "archive.zip"
const cacheArchiverTestArchivedFile = "archive_file"

func TestCacheArchiverIsUpToDate(t *testing.T) {
	ioutil.WriteFile(cacheArchiverTestArchivedFile, nil, 0600)
	defer os.Remove(cacheArchiverTestArchivedFile)

	defer os.Remove(cacheArchiverArchive)
	cmd := CacheArchiverCommand{
		File: cacheArchiverArchive,
		fileArchiver: fileArchiver{
			Paths: []string{
				cacheArchiverTestArchivedFile,
			},
		},
	}
	cmd.Execute(nil)
	fi, _ := os.Stat(cacheArchiverArchive)
	cmd.Execute(nil)
	fi2, _ := os.Stat(cacheArchiverArchive)
	assert.Equal(t, fi.ModTime(), fi2.ModTime())

	os.Chtimes(cacheArchiverTestArchivedFile, time.Now(), time.Now())
	cmd.Execute(nil)
	fi3, _ := os.Stat(cacheArchiverArchive)
	assert.Equal(t, fi.ModTime(), fi3.ModTime())
}

func TestCacheArchiverForIfNoFileDefined(t *testing.T) {
	helpers.MakeFatalToPanic()
	cmd := CacheArchiverCommand{}
	assert.Panics(t, func() {
		cmd.Execute(nil)
	})
}

func TestCacheArchiverForNotExistingFile(t *testing.T) {
	helpers.MakeFatalToPanic()
	cmd := CacheArchiverCommand{
		File: "/../../../test.zip",
	}
	assert.Panics(t, func() {
		cmd.Execute(nil)
	})
}
