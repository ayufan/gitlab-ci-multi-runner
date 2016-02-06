package helpers

import "github.com/Sirupsen/logrus"

type fatalLogHook struct {
}

func (s *fatalLogHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.FatalLevel,
	}
}

func (s *fatalLogHook) Fire(e *logrus.Entry) error {
	panic(e)
	return nil
}

func MakeFatalToPanic() {
	logrus.AddHook(&fatalLogHook{})
}
