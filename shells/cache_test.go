package shells

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

var s3Cache = common.CacheConfig{
	Type:           "s3",
	ServerAddress:  "server.com",
	AccessKey:      "access",
	SecretKey:      "key",
	BucketName:     "test",
	BucketLocation: "location",
}

var s3CacheBuild = &common.Build{
	GetBuildResponse: common.GetBuildResponse{
		ProjectID: 10,
		Timeout:   3600,
	},
	Runner: &common.RunnerConfig{
		RunnerCredentials: common.RunnerCredentials{
			Token: "longtoken",
		},
		RunnerSettings: common.RunnerSettings{
			Cache: &s3Cache,
		},
	},
}

func TestS3CacheUploadURL(t *testing.T) {
	url := getCacheUploadURL(s3CacheBuild, "key")
	require.NotNil(t, url)
	assert.Equal(t, s3Cache.ServerAddress, url.Host)
}

func TestS3CacheDownloadURL(t *testing.T) {
	url := getCacheDownloadURL(s3CacheBuild, "key")
	require.NotNil(t, url)
	assert.Equal(t, s3Cache.ServerAddress, url.Host)
}
