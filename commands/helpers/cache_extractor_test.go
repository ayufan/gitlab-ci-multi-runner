package helpers

import (
	"archive/zip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"time"
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

func testServeCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "408 Method not allowed", 408)
		return
	}
	if r.URL.Path != "/cache.zip" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
	archive := zip.NewWriter(w)
	archive.Create(cacheExtractorTestArchivedFile)
	archive.Close()
}

func TestCacheExtractorRemoteServerNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testServeCache))
	defer ts.Close()

	helpers.MakeFatalToPanic()
	cmd := CacheExtractorCommand{
		File: "non-existing-test.zip",
		URL:  ts.URL + "/invalid-file.zip",
	}
	assert.NotPanics(t, func() {
		cmd.Execute(nil)
	})
	_, err := os.Stat(cacheExtractorTestArchivedFile)
	assert.Error(t, err)
}

func TestCacheExtractorRemoteServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testServeCache))
	defer ts.Close()

	defer os.Remove(cacheExtractorArchive)
	defer os.Remove(cacheExtractorTestArchivedFile)
	os.Remove(cacheExtractorArchive)
	os.Remove(cacheExtractorTestArchivedFile)

	helpers.MakeFatalToPanic()
	cmd := CacheExtractorCommand{
		File: cacheExtractorArchive,
		URL:  ts.URL + "/cache.zip",
	}
	assert.NotPanics(t, func() {
		cmd.Execute(nil)
	})

	_, err := os.Stat(cacheExtractorTestArchivedFile)
	assert.NoError(t, err)

	os.Chtimes(cacheExtractorArchive, time.Now().Add(time.Hour), time.Now().Add(time.Hour))
	assert.NotPanics(t, func() {
		cmd.Execute(nil)
	}, "archive is up to date")
}

func TestCacheExtractorRemoteServerDoesntFailOnInvalidServer(t *testing.T) {
	helpers.MakeFatalToPanic()
	os.Remove(cacheExtractorArchive)
	cmd := CacheExtractorCommand{
		File: cacheExtractorArchive,
		URL:  "http://localhost:65333/cache.zip",
	}
	assert.NotPanics(t, func() {
		cmd.Execute(nil)
	})

	_, err := os.Stat(cacheExtractorTestArchivedFile)
	assert.Error(t, err)
}
