package commands_helpers

import (
	"archive/zip"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const UntrackedFileName = "some_fancy_untracked_file"

var currentDir, _ = os.Getwd()

func randomTempFile(t *testing.T, format string) string {
	file, err := ioutil.TempFile("", "archive_")
	assert.NoError(t, err)
	defer file.Close()
	defer os.Remove(file.Name())
	return file.Name() + format
}

func createArchiveCommand(t *testing.T) *ArchiveCommand {
	err := os.Chdir(filepath.Join(currentDir, "..", ".."))
	assert.NoError(t, err)

	return &ArchiveCommand{
		File:    randomTempFile(t, ".zip"),
		Verbose: true,
	}
}

func filesInFolder(path string) []string {
	matches, _ := filepath.Glob(path)
	return matches
}

func readArchiveContent(t *testing.T, c *ArchiveCommand) (resultMap map[string]bool) {
	resultMap = make(map[string]bool)

	archive, err := zip.OpenReader(c.File)
	assert.NoError(t, err)
	defer archive.Close()
	for _, file := range archive.File {
		resultMap[file.Name] = true
	}
	return
}

func verifyArchiveContent(t *testing.T, c *ArchiveCommand, files ...string) {
	resultMap := readArchiveContent(t, c)
	for _, file := range files {
		assert.True(t, resultMap[file], "File should exist %q", file)
		delete(resultMap, file)
	}
	assert.Len(t, resultMap, 0, "No extra file should exist")
}

func TestArchiveNotCreatingArchive(t *testing.T) {
	cmd := createArchiveCommand(t)
	defer os.Remove(cmd.File)
	cmd.Execute(nil)
	_, err := os.Stat(cmd.File)
	assert.True(t, os.IsNotExist(err), "File should not exist", cmd.File, err)
}

func TestArchiveAddingSomeLocalFiles(t *testing.T) {
	cmd := createArchiveCommand(t)
	defer os.Remove(cmd.File)
	cmd.Paths = []string{
		"commands/helpers/*",
	}
	cmd.Execute(nil)
	verifyArchiveContent(t, cmd, filesInFolder("commands/helpers/*")...)
}

func TestArchiveNotAddingDuplicateFiles(t *testing.T) {
	cmd := createArchiveCommand(t)
	defer os.Remove(cmd.File)
	cmd.Paths = []string{
		"commands/helpers/*",
		"commands/helpers/archive.go",
	}
	cmd.Execute(nil)
	verifyArchiveContent(t, cmd, filesInFolder("commands/helpers/*")...)
}

func TestArchiveAddingUntrackedFiles(t *testing.T) {
	cmd := createArchiveCommand(t)
	defer os.Remove(cmd.File)
	err := ioutil.WriteFile(UntrackedFileName, []byte{}, 0700)
	assert.NoError(t, err)
	cmd.Untracked = true
	cmd.Execute(nil)
	files := readArchiveContent(t, cmd)
	assert.NotEmpty(t, files)
	assert.True(t, files[UntrackedFileName])
}

func TestArchiveUpdating(t *testing.T) {
	tempFile := randomTempFile(t, ".zip")
	defer os.Remove(tempFile)

	err := ioutil.WriteFile(UntrackedFileName, []byte{}, 0700)
	assert.NoError(t, err)

	cmd := createArchiveCommand(t)
	defer os.Remove(cmd.File)
	cmd.Paths = []string{
		"commands",
		UntrackedFileName,
	}

	cmd.Execute(nil)
	archive1, err := os.Stat(cmd.File)
	assert.NoError(t, err, "Archive is created")

	cmd.Execute(nil)
	archive2, err := os.Stat(cmd.File)
	assert.NoError(t, err, "Archive is created")
	assert.Equal(t, archive1.ModTime(), archive2.ModTime(), "Archive should not be modified")

	time.Sleep(time.Second)
	err = ioutil.WriteFile(UntrackedFileName, []byte{}, 0700)
	assert.NoError(t, err, "File is created")

	cmd.Execute(nil)
	archive3, err := os.Stat(cmd.File)
	assert.NoError(t, err, "Archive is created")
	assert.NotEqual(t, archive2.ModTime(), archive3.ModTime(), "File is added to archive")

	time.Sleep(time.Second)
	err = ioutil.WriteFile(UntrackedFileName, []byte{}, 0700)
	assert.NoError(t, err, "File is updated")

	cmd.Execute(nil)
	archive4, err := os.Stat(cmd.File)
	assert.NoError(t, err, "Archive is created")
	assert.NotEqual(t, archive3.ModTime(), archive4.ModTime(), "File is updated in archive")
}
