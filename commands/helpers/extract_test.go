package commands_helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/EMSSConsulting/Thargo"
	"github.com/stretchr/testify/assert"
)

func createExtractCommand(t *testing.T) *ExtractCommand {
	err := os.Chdir(filepath.Join(currentDir, ".."))
	assert.NoError(t, err)

	return &ExtractCommand{
		File: randomTempFile(t),
	}
}

func writeArchive(t *testing.T, c *ExtractCommand, target thargo.Target) {
	options := *thargo.DefaultOptions
	options.CreateIfMissing = true
	archive, err := thargo.NewArchiveFile(c.File, &options)
	assert.NoError(t, err)
	defer archive.Close()
	assert.NoError(t, archive.Add(target))
}

func TestExtract(t *testing.T) {
	c := createExtractCommand(t)
	writeArchive(t, c, &thargo.StringTarget{
		Name:    "test.txt",
		Content: "test file",
	})
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
