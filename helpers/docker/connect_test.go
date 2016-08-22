package docker_helpers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/docker"
)

func TestCachesClient(t *testing.T) {
	client1, err := docker_helpers.New(docker_helpers.DockerCredentials{}, "1.0")
	require.NoError(t, err)

	client2, err := docker_helpers.New(docker_helpers.DockerCredentials{}, "1.0")
	require.NoError(t, err)

	assert.Equal(t, client1, client2)
}

func TestNoCacheForDifferentCredentials(t *testing.T) {
	client1, err := docker_helpers.New(docker_helpers.DockerCredentials{}, "1.1")
	require.NoError(t, err)

	client2, err := docker_helpers.New(docker_helpers.DockerCredentials{Host: "google.com"}, "1.1")
	require.NoError(t, err)

	assert.NotEqual(t, client1, client2)
}

func TestCacheUserUnixSockets(t *testing.T) {
	dc := docker_helpers.DockerCredentials{Host: "unix://google.com/"}

	client1, err := docker_helpers.New(dc, "1.1")
	require.NoError(t, err)

	client2, err := docker_helpers.New(dc, "1.1")
	require.NoError(t, err)

	assert.Equal(t, client1, client2)
}

func TestNoCacheNonUnixSockets(t *testing.T) {
	dc := docker_helpers.DockerCredentials{Host: "http://google.com/"}

	client1, err := docker_helpers.New(dc, "1.1")
	require.NoError(t, err)

	client2, err := docker_helpers.New(dc, "1.1")
	require.NoError(t, err)

	assert.NotEqual(t, client1, client2)
}
