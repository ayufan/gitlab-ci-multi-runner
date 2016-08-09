package docker_helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCachesClient(t *testing.T) {
	assert.Equal(t, DockerCredentials{}, DockerCredentials{}, "OK")
	client1, err := New(DockerCredentials{}, "1.0")
	assert.NoError(t, err)

	client2, err := New(DockerCredentials{}, "1.0")
	assert.NoError(t, err)

	assert.Equal(t, client1, client2, "New() with identical args should be cached")

	client3, err := New(DockerCredentials{}, "1.1")
	assert.NoError(t, err)

	client4, err := New(DockerCredentials{Host: "google.com"}, "1.1")
	assert.NoError(t, err)

	assert.NotEqual(t, client3, client4, "New() with differing args should not be cached")
}
