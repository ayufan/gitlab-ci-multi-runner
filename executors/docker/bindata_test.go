package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrebuiltX86_64Assets(t *testing.T) {
	_, err := Asset("prebuilt-x86_64" + prebuiltImageExtension)
	assert.NoError(t, err)
}

func TestPrebuiltARMAssets(t *testing.T) {
	_, err := Asset("prebuilt-arm" + prebuiltImageExtension)
	assert.NoError(t, err)
}
