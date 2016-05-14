package shells

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/minio/minio-go"

	"github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type bucketLocationTripper struct {
	bucketLocation string
}

func (b *bucketLocationTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	var buffer bytes.Buffer
	xml.NewEncoder(&buffer).Encode(b.bucketLocation)
	res = &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(&buffer),
	}
	return
}

func (b *bucketLocationTripper) CancelRequest(req *http.Request) {
	// Do nothing
}

func getCacheObjectName(build *common.Build, cache *common.CacheConfig, key string) string {
	if key == "" {
		return ""
	}
	return path.Join("runner", build.Runner.ShortDescription(), "project", strconv.Itoa(build.ProjectID), key)
}

func getCacheStorageClient(cache *common.CacheConfig) (scl *minio.Client, err error) {
	scl, err = minio.New(cache.ServerAddress, cache.AccessKey, cache.SecretKey, cache.Insecure)
	if err != nil {
		logrus.Warningln(err)
		return
	}

	scl.SetCustomTransport(&bucketLocationTripper{cache.BucketLocation})
	return
}

func getS3DownloadURL(build *common.Build, key string) (url *url.URL) {
	cache := build.Runner.Cache
	objectName := getCacheObjectName(build, cache, key)
	if objectName == "" {
		return
	}

	scl, err := getCacheStorageClient(cache)
	if err != nil {
		logrus.Warningln(err)
		return
	}

	url, err = scl.PresignedGetObject(cache.BucketName, objectName, time.Second*time.Duration(build.Timeout), nil)
	if err != nil {
		logrus.Warningln(err)
		return
	}
	return
}

func getCacheDownloadURL(build *common.Build, key string) (url *url.URL) {
	cache := build.Runner.Cache
	if cache == nil {
		return
	}

	switch cache.Type {
	case "s3":
		return getS3DownloadURL(build, key)
	}
	return
}

func getS3UploadURL(build *common.Build, key string) (url *url.URL) {
	cache := build.Runner.Cache
	objectName := getCacheObjectName(build, cache, key)
	if objectName == "" {
		return
	}

	scl, err := getCacheStorageClient(cache)
	if err != nil {
		logrus.Warningln(err)
		return
	}

	url, err = scl.PresignedPutObject(cache.BucketName, objectName, time.Second*time.Duration(build.Timeout))
	if err != nil {
		logrus.Warningln(err)
		return
	}
	return
}

func getCacheUploadURL(build *common.Build, key string) (url *url.URL) {
	cache := build.Runner.Cache
	if cache == nil {
		return
	}

	switch cache.Type {
	case "s3":
		return getS3UploadURL(build, key)
	}
	return
}
