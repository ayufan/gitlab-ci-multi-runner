package docker

import (
	"bytes"
	"compress/gzip"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"testing"
)

func TestPrebuiltX86_64Assets(t *testing.T) {
	data, err := Asset("prebuilt-x86_64.tar.gz")
	assert.NoError(t, err)

	gz, err := gzip.NewReader(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.NotNil(t, gz)
	defer gz.Close()

	_, err = io.Copy(ioutil.Discard, gz)
	assert.NoError(t, err)
}

func TestPrebuiltARMAssets(t *testing.T) {
	data, err := Asset("prebuilt-arm.tar.gz")
	assert.NoError(t, err)

	gz, err := gzip.NewReader(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.NotNil(t, gz)
	defer gz.Close()

	_, err = io.Copy(ioutil.Discard, gz)
	assert.NoError(t, err)
}
