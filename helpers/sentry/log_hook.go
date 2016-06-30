package sentry

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/getsentry/raven-go"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type LogHook struct {
	client *raven.Client
}

func (s *LogHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
	}
}

func (s *LogHook) Fire(entry *logrus.Entry) error {
	if s.client == nil {
		return nil
	}

	tags := make(map[string]string)
	for key, value := range entry.Data {
		tags[key] = fmt.Sprint(value)
	}

	switch entry.Level {
	case logrus.PanicLevel:
		s.client.CaptureErrorAndWait(errors.New(entry.Message), tags)

	case logrus.FatalLevel:
		s.client.CaptureErrorAndWait(errors.New(entry.Message), tags)

	case logrus.ErrorLevel:
		s.client.CaptureError(errors.New(entry.Message), tags)
	}
	return nil
}

func NewLogHook(dsn string) (lh LogHook, err error) {
	tags := make(map[string]string)
	tags["built"] = common.BUILT
	tags["version"] = common.VERSION
	tags["revision"] = common.REVISION
	tags["branch"] = common.BRANCH
	tags["go-version"] = runtime.Version()
	tags["go-os"] = runtime.GOOS
	tags["go-arch"] = runtime.GOARCH
	tags["hostname"], _ = os.Hostname()
	client, err := raven.NewWithTags(dsn, tags)
	if err != nil {
		return
	}
	lh.client = client
	return
}
