package helpers

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"time"
)

const fileArchiverUntrackedFile = "untracked_test_file.txt"
const fileArchiverOtherFile = "other_test_file.txt"
const fileArchiverNotExistingFile = "not_existing_file.txt"
const fileArchiverAbsoluteFile = "/absolute.txt"
const fileArchiverRelativeFile = "../../../relative.txt"

func TestCacheArchiverAddingUntrackedFiles(t *testing.T) {
	ioutil.WriteFile(fileArchiverUntrackedFile, nil, 0600)
	defer os.Remove(fileArchiverUntrackedFile)

	f := fileArchiver{
		Untracked: true,
	}
	err := f.enumerate()
	assert.NoError(t, err)
	assert.Len(t, f.sortedFiles(), 1)
	assert.Contains(t, f.sortedFiles(), fileArchiverUntrackedFile)
}

func TestCacheArchiverAddingFile(t *testing.T) {
	ioutil.WriteFile(fileArchiverUntrackedFile, nil, 0600)
	defer os.Remove(fileArchiverUntrackedFile)

	f := fileArchiver{
		Paths: []string{fileArchiverUntrackedFile},
	}
	err := f.enumerate()
	assert.NoError(t, err)
	assert.Len(t, f.sortedFiles(), 1)
	assert.Contains(t, f.sortedFiles(), fileArchiverUntrackedFile)
}

func TestFileArchiverToFailOnAbsoulteFile(t *testing.T) {
	f := fileArchiver{
		Paths: []string{fileArchiverAbsoluteFile},
	}
	err := f.enumerate()
	assert.NoError(t, err)
	assert.Empty(t, f.sortedFiles())
	assert.NotContains(t, f.sortedFiles(), fileArchiverAbsoluteFile)
}

func TestFileArchiverToFailOnRelativeFile(t *testing.T) {
	f := fileArchiver{
		Paths: []string{fileArchiverRelativeFile},
	}
	err := f.enumerate()
	assert.NoError(t, err)
	assert.Empty(t, f.sortedFiles())
}

func TestFileArchiverToAddNotExistingFile(t *testing.T) {
	f := fileArchiver{
		Paths: []string{fileArchiverNotExistingFile},
	}
	err := f.enumerate()
	assert.NoError(t, err)
	assert.Empty(t, f.sortedFiles())
}

func TestFileArchiverChanged(t *testing.T) {
	ioutil.WriteFile(fileArchiverOtherFile, nil, 0600)
	defer os.Remove(fileArchiverOtherFile)

	ioutil.WriteFile(fileArchiverUntrackedFile, nil, 0600)
	defer os.Remove(fileArchiverUntrackedFile)

	f := fileArchiver{
		Paths: []string{fileArchiverUntrackedFile},
	}
	err := f.enumerate()
	assert.NoError(t, err)
	assert.Len(t, f.sortedFiles(), 1)
	assert.False(t, f.isChanged(time.Now().Add(time.Minute)))
	assert.True(t, f.isChanged(time.Now().Add(-time.Minute)))

	assert.False(t, f.isFileChanged(fileArchiverOtherFile), "should return false if file was modified before the listed file")
	os.Chtimes(fileArchiverOtherFile, time.Now(), time.Now().Add(-time.Minute))
	assert.True(t, f.isFileChanged(fileArchiverOtherFile), "should return true if file was modified")
	assert.True(t, f.isFileChanged(fileArchiverNotExistingFile), "should return true if file doesn't exist")
}
