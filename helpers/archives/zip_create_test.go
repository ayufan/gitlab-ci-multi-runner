package archives

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testZipFileContent = []byte("test content")

func createTestFile(t *testing.T) string {
	err := ioutil.WriteFile("test_file.txt", testZipFileContent, 0640)
	assert.NoError(t, err)
	return "test_file.txt"
}

func createSymlinkFile(t *testing.T) string {
	err := os.Symlink("old_symlink", "new_symlink")
	assert.NoError(t, err)
	return "new_symlink"
}

func createTestDirectory(t *testing.T) string {
	err := os.Mkdir("test_directory", 0711)
	assert.NoError(t, err)
	return "test_directory"
}

func createTestPipe(t *testing.T) string {
	err := syscall.Mkfifo("test_pipe", 0600)
	assert.NoError(t, err)
	return "test_pipe"
}

func TestZipCreate(t *testing.T) {
	td, err := ioutil.TempDir("", "zip_create")
	if !assert.NoError(t, err) {
		return
	}

	wd, err := os.Getwd()
	assert.NoError(t, err)
	defer os.Chdir(wd)

	err = os.Chdir(td)
	assert.NoError(t, err)

	tempFile, err := ioutil.TempFile("", "archive")
	if !assert.NoError(t, err) {
		return
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	err = CreateZipFile(tempFile.Name(), []string{
		createTestFile(t),
		createSymlinkFile(t),
		createTestDirectory(t),
		createTestPipe(t),
		"non_existing_file.txt",
	})
	if !assert.NoError(t, err) {
		return
	}

	archive, err := zip.OpenReader(tempFile.Name())
	if !assert.NoError(t, err) {
		return
	}
	defer archive.Close()

	assert.Len(t, archive.File, 3)
	assert.Equal(t, "test_file.txt", archive.File[0].Name)
	assert.Equal(t, 0640, archive.File[0].Mode().Perm())
	assert.NotEmpty(t, archive.File[0].Extra)
	assert.Equal(t, "new_symlink", archive.File[1].Name)
	assert.Equal(t, "test_directory/", archive.File[2].Name)
	assert.NotEmpty(t, archive.File[2].Extra)
	assert.True(t, archive.File[2].Mode().IsDir())
}
