package archives

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func writeArchive(t *testing.T, w io.Writer) {
	archive := zip.NewWriter(w)
	defer archive.Close()

	testFile, err := archive.Create("temporary_file.txt")
	if !assert.NoError(t, err) {
		return
	}
	io.WriteString(testFile, "test file")
}

func TestExtractZipFile(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "archive")
	if !assert.NoError(t, err) {
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())
	writeArchive(t, tempFile)
	tempFile.Close()

	err = ExtractZipFile(tempFile.Name())
	if !assert.NoError(t, err) {
		return
	}

	stat, err := os.Stat("temporary_file.txt")
	assert.False(t, os.IsNotExist(err), "Expected temporary_file.txt to exist")
	if !os.IsNotExist(err) {
		assert.NoError(t, err)
	}

	if stat != nil {
		defer os.Remove("temporary_file.txt")
		assert.Equal(t, int64(9), stat.Size())
	}
}

func TestExtractZipFileNotFound(t *testing.T) {
	err := ExtractZipFile("non_existing_zip_file.zip")
	assert.Error(t, err)
}
