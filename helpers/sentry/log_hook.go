package sentry

import (
	"fmt"
	"errors"

	"github.com/Sirupsen/logrus"
	"github.com/getsentry/raven-go"
	"time"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"runtime"
	"os"
)

type LogHook struct {
	client *raven.Client
}

func (s *LogHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
	}
}

func (s *LogHook) Fire(entry *logrus.Entry) error {
	if s.client == nil {
		return
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

	case logrus.WarnLevel:
		s.client.CaptureMessage(entry.Message, tags)
	}
}

func NewLogHook(dsn string) *LogHook {
	tags := make(map[string]string)

	tags["built"] = common.BUILT
	tags["version"] = common.VERSION
	tags["revision"] = common.REVISION
	tags["branch"] = common.BRANCH
	tags["go-version"] = runtime.Version()
	tags["go-os"] = runtime.GOOS
	tags["go-arch"] = runtime.GOARCH
	tags["hostname"], _ = os.Hostname()

	return &LogHook{
		client: raven.NewWithTags(dsn, tags)
	}
}
