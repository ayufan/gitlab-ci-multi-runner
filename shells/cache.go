package shells

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"net/http"
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

func getCacheObjectName(build *common.Build, cache *common.CacheConfig, key string) string {
	if key == "" || cache == nil || !cache.UseS3 {
		return ""
	}
	return path.Join("runner", build.Runner.ShortDescription(), "project", strconv.Itoa(build.ProjectID), key)
}

func getCacheStorageClient(cache *common.CacheConfig) (scl minio.CloudStorageClient, err error) {
	scl, err = minio.New(cache.ServerAddress, cache.AccessKey, cache.SecretKey, cache.Insecure)
	if err != nil {
		logrus.Warningln(err)
		return
	}

	scl.SetCustomTransport(&bucketLocationTripper{cache.BucketLocation})
	return
}

func getCacheDownloadURL(build *common.Build, key string) (url string) {
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

	url, err = scl.PresignedGetObject(cache.BucketName, key, time.Second*time.Duration(build.Timeout))
	if err != nil {
		logrus.Warningln(err)
		return
	}

	return
}

func getCacheUploadURL(build *common.Build, key string) (url string) {
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

	url, err = scl.PresignedPutObject(cache.BucketName, key, time.Second*time.Duration(build.Timeout))
	if err != nil {
		logrus.Warningln(err)
		return
	}

	return
}
