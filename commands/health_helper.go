package commands

import (
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type healthData struct {
	failures  int
	lastCheck time.Time
}

type healthHelper struct {
	healthy     map[string]*healthData
	healthyLock sync.Mutex
}

func (mr *healthHelper) getHealth(id string) *healthData {
	mr.healthyLock.Lock()
	defer mr.healthyLock.Unlock()

	if mr.healthy == nil {
		mr.healthy = map[string]*healthData{}
	}
	health := mr.healthy[id]
	if health == nil {
		health = &healthData{
			lastCheck: time.Now(),
		}
		mr.healthy[id] = health
	}
	return health
}

func (mr *healthHelper) isHealthy(id string) bool {
	health := mr.getHealth(id)
	if health.failures < common.HealthyChecks {
		return true
	}

	if time.Since(health.lastCheck) > common.HealthCheckInterval*time.Second {
		logrus.Errorln("Runner", id, "is not healthy, but will be checked!")
		health.failures = 0
		health.lastCheck = time.Now()
		return true
	}

	return false
}

func (mr *healthHelper) makeHealthy(id string, healthy bool) {
	health := mr.getHealth(id)
	if healthy {
		health.failures = 0
		health.lastCheck = time.Now()
	} else {
		health.failures++
		if health.failures >= common.HealthyChecks {
			logrus.Errorln("Runner", id, "is not healthy and will be disabled!")
		}
	}
}
