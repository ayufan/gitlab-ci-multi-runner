package commands_helpers

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"io"
)

func createExtractCommand(t *testing.T) *ExtractCommand {
	err := os.Chdir(filepath.Join(currentDir, "..", ".."))
	assert.NoError(t, err)

	return &ExtractCommand{
		File: randomTempFile(t, ".zip"),
	}
}

func writeArchive(t *testing.T, c *ExtractCommand) {
	file, err := os.Create(c.File)
	if !assert.NoError(t, err) {
		return
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	defer archive.Close()

	testFile, err := archive.Create("test.txt")
	if !assert.NoError(t, err) {
		return
	}
	io.WriteString(testFile, "test file")
}

func TestExtract(t *testing.T) {
	c := createExtractCommand(t)
	writeArchive(t, c)
	defer os.Remove(c.File)

	c.Execute(nil)

	stat, err := os.Stat("test.txt")
	assert.False(t, os.IsNotExist(err), "Expected test.txt to exist")
	if !os.IsNotExist(err) {
		assert.NoError(t, err)
	}

	if stat != nil {
		defer os.Remove("test.txt")
		assert.Equal(t, int64(9), stat.Size())
	}
}
