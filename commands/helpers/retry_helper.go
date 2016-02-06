package helpers

import (
	"github.com/Sirupsen/logrus"
	"time"
)

type retryHelper struct {
	Retry     int           `long:"retry" description:"How many times to retry upload"`
	RetryTime time.Duration `long:"retry-time" description:"How long to wait between retries"`
}

func (r *retryHelper) doRetry(handler func() (bool, error)) (err error) {
	retry, err := handler()
	for i := 0; retry && i < r.Retry; i++ {
		// wait one second to retry
		logrus.Warningln("Retrying...")
		time.Sleep(r.RetryTime)
		retry, err = handler()
	}
	return
}
