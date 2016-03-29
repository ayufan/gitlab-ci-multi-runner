package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
	assert.Equal(t, fi.ModTime(), fi2.ModTime(), "archive is up to date")

	// We need to wait one second, since the FS doesn't save milliseconds
	time.Sleep(time.Second)

	os.Chtimes(cacheArchiverTestArchivedFile, time.Now(), time.Now())
	cmd.Execute(nil)
	fi3, _ := os.Stat(cacheArchiverArchive)
	assert.NotEqual(t, fi.ModTime(), fi3.ModTime(), "archive should get updated")
}

func TestCacheArchiverForIfNoFileDefined(t *testing.T) {
	helpers.MakeFatalToPanic()
	cmd := CacheArchiverCommand{}
	assert.Panics(t, func() {
		cmd.Execute(nil)
	})
}

func testCacheUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "408 Method not allowed", 408)
		return
	}
	if r.URL.Path != "/cache.zip" {
		http.NotFound(w, r)
		return
	}
}

func TestCacheArchiverRemoteServerNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testCacheUploadHandler))
	defer ts.Close()

	helpers.MakeFatalToPanic()
	os.Remove(cacheExtractorArchive)
	cmd := CacheArchiverCommand{
		File: cacheExtractorArchive,
		URL:  ts.URL + "/invalid-file.zip",
	}
	assert.NotPanics(t, func() {
		cmd.Execute(nil)
	})
}

func TestCacheArchiverRemoteServe(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testCacheUploadHandler))
	defer ts.Close()

	helpers.MakeFatalToPanic()
	os.Remove(cacheExtractorArchive)
	cmd := CacheArchiverCommand{
		File: cacheExtractorArchive,
		URL:  ts.URL + "/cache.zip",
	}
	assert.NotPanics(t, func() {
		cmd.Execute(nil)
	})
}

func TestCacheArchiverRemoteServerDoesntFailOnInvalidServer(t *testing.T) {
	helpers.MakeFatalToPanic()
	os.Remove(cacheExtractorArchive)
	cmd := CacheArchiverCommand{
		File: cacheExtractorArchive,
		URL:  "http://localhost:65333/cache.zip",
	}
	assert.NotPanics(t, func() {
		cmd.Execute(nil)
	})

	_, err := os.Stat(cacheExtractorTestArchivedFile)
	assert.Error(t, err)
}
