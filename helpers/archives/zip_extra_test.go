package archives

import (
	"io/ioutil"
	"os"
	"testing"

	"archive/zip"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
)

func TestCreateZipExtra(t *testing.T) {
	testFile, err := ioutil.TempFile("", "test")
	assert.NoError(t, err)
	defer testFile.Close()
	defer os.Remove(testFile.Name())

	fi, _ := testFile.Stat()
	assert.NotNil(t, fi)

	data := createZipExtra(fi)
	assert.NotEmpty(t, data)
	assert.Len(t, data, binary.Size(&ZipExtraField{})*2+
		binary.Size(&ZipUIDGidField{})+
		binary.Size(&ZipTimestampField{}))
}

func TestProcessZipExtra(t *testing.T) {
	testFile, err := ioutil.TempFile("", "test")
	assert.NoError(t, err)
	defer testFile.Close()
	defer os.Remove(testFile.Name())

	fi, _ := testFile.Stat()
	assert.NotNil(t, fi)

	zipFile, err := zip.FileInfoHeader(fi)
	assert.NoError(t, err)
	zipFile.Extra = createZipExtra(fi)

	err = ioutil.WriteFile(fi.Name(), []byte{}, 0666)
	defer os.Remove(fi.Name())
	assert.NoError(t, err)

	err = processZipExtra(zipFile)
	assert.NoError(t, err)

	fi2, _ := testFile.Stat()
	assert.NotNil(t, fi2)
	assert.Equal(t, fi.Mode(), fi2.Mode())
	assert.Equal(t, fi.ModTime(), fi2.ModTime())
}
