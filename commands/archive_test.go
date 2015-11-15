package commands

import (
	"bufio"
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"strings"
	"time"
)

var currentDir, _ = os.Getwd()

func randomTempFile(t *testing.T) string {
	file, err := ioutil.TempFile("", "archive_")
	assert.NoError(t, err)
	defer file.Close()
	defer os.Remove(file.Name())
	return file.Name()
}

func createArchiveCommand(t *testing.T) *ArchiveCommand {
	err := os.Chdir(filepath.Join(currentDir, ".."))
	assert.NoError(t, err)

	return &ArchiveCommand{
		Output: randomTempFile(t),
		Silent: true,
	}
}

func filesInFolder(path string) []string {
	matches, _ := filepath.Glob(path)
	return matches
}

func readArchiveContent(t *testing.T, c *ArchiveCommand) (resultMap map[string]bool) {
	resultMap = make(map[string]bool)

	cmd := exec.Command("tar", "-ztf", c.Output)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	result, err := cmd.Output()
	assert.NoError(t, err)

	reader := bufio.NewReader(bytes.NewReader(result))
	for {
		line, prefix, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		assert.NoError(t, err, "ReadLine")
		assert.False(t, prefix, "ReadLine")
		file := strings.TrimSpace(string(line))
		resultMap[file] = true
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
	defer os.Remove(cmd.Output)
	cmd.Execute(nil)
	_, err := os.Stat(cmd.Output)
	assert.True(t, os.IsNotExist(err), "File should not exist", cmd.Output, err)
}

func TestArchiveAddingSomeLocalFiles(t *testing.T) {
	cmd := createArchiveCommand(t)
	defer os.Remove(cmd.Output)
	cmd.Paths = []string{
		"commands/*",
	}
	cmd.Execute(nil)
	verifyArchiveContent(t, cmd, filesInFolder("commands/*")...)
}

func TestArchiveNotAddingDuplicateFiles(t *testing.T) {
	cmd := createArchiveCommand(t)
	defer os.Remove(cmd.Output)
	cmd.Paths = []string{
		"commands/*",
		"commands/archive.go",
	}
	cmd.Execute(nil)
	verifyArchiveContent(t, cmd, filesInFolder("commands/*")...)
}

func TestArchiveAddingUntrackedFiles(t *testing.T) {
	cmd := createArchiveCommand(t)
	defer os.Remove(cmd.Output)
	cmd.Untracked = true
	cmd.Execute(nil)
	files := readArchiveContent(t, cmd)
	assert.NotEmpty(t, files)
}

func TestArchiveUpdating(t *testing.T) {
	tempFile := randomTempFile(t)
	defer os.Remove(tempFile)

	cmd := createArchiveCommand(t)
	defer os.Remove(cmd.Output)
	cmd.Paths = []string{
		"commands",
		tempFile,
	}

	cmd.Execute(nil)
	archive1, err := os.Stat(cmd.Output)
	assert.NoError(t, err, "Archive is created")

	cmd.Execute(nil)
	archive2, err := os.Stat(cmd.Output)
	assert.NoError(t, err, "Archive is created")
	assert.Equal(t, archive1.ModTime(), archive2.ModTime(), "Archive should not be modyfied")

	time.Sleep(time.Second)
	err = ioutil.WriteFile(tempFile, []byte{}, 0700)
	assert.NoError(t, err, "File is created")

	cmd.Execute(nil)
	archive3, err := os.Stat(cmd.Output)
	assert.NoError(t, err, "Archive is created")
	assert.NotEqual(t, archive2.ModTime(), archive3.ModTime(), "File is added to archive")

	time.Sleep(time.Second)
	err = ioutil.WriteFile(tempFile, []byte{}, 0700)
	assert.NoError(t, err, "File is updated")

	cmd.Execute(nil)
	archive4, err := os.Stat(cmd.Output)
	assert.NoError(t, err, "Archive is created")
	assert.NotEqual(t, archive3.ModTime(), archive4.ModTime(), "File is updated in archive")
}

