package network

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/mocks"
	"testing"
	"time"
)

const successID = 4
const cancelID = 5
const retryID = 6

var buildConfig = common.RunnerConfig{}
var buildOutputLimit = common.RunnerConfig{OutputLimit: 1}

type updateTraceNetwork struct {
	mocks.Network
	state common.BuildState
	trace *string
	count int
}

func (m *updateTraceNetwork) UpdateBuild(config common.RunnerConfig, id int, state common.BuildState, trace *string) common.UpdateState {
	switch id {
	case successID:
		m.count++
		m.state = state
		m.trace = trace
		return common.UpdateSucceeded

	case cancelID:
		m.count++
		return common.UpdateAbort

	case retryID:
		if state != common.Running {
			m.count++
			if m.count >= 5 {
				m.state = state
				m.trace = trace
				return common.UpdateSucceeded
			}
		}
		return common.UpdateFailed

	default:
		return common.UpdateFailed
	}
}

func (m *updateTraceNetwork) PatchTrace(config common.RunnerConfig, buildCredentials *common.BuildCredentials, tracePatch common.BuildTracePatch) common.UpdateState {
	return common.UpdateNotFound
}

func TestBuildTraceSuccess(t *testing.T) {
	u := &updateTraceNetwork{}
	buildCredentials := &common.BuildCredentials{
		ID: successID,
	}
	b := newBuildTrace(u, buildConfig, buildCredentials)
	b.start()
	fmt.Fprint(b, "test content")
	b.Success()
	assert.Equal(t, "test content", *u.trace)
	assert.Equal(t, common.Success, u.state)
}

func TestBuildTraceFailure(t *testing.T) {
	u := &updateTraceNetwork{}
	buildCredentials := &common.BuildCredentials{
		ID: successID,
	}
	b := newBuildTrace(u, buildConfig, buildCredentials)
	b.start()
	fmt.Fprint(b, "test content")
	b.Fail(errors.New("test"))
	assert.Equal(t, "test content", *u.trace)
	assert.Equal(t, common.Failed, u.state)
}

func TestIgnoreStatusChange(t *testing.T) {
	u := &updateTraceNetwork{}
	buildCredentials := &common.BuildCredentials{
		ID: successID,
	}
	b := newBuildTrace(u, buildConfig, buildCredentials)
	b.start()
	b.Success()
	b.Fail(errors.New("test"))
	assert.Equal(t, common.Success, u.state)
}

func TestBuildAbort(t *testing.T) {
	traceUpdateInterval = 0

	abort := make(chan bool)

	u := &updateTraceNetwork{}
	buildCredentials := &common.BuildCredentials{
		ID: cancelID,
	}
	b := newBuildTrace(u, buildConfig, buildCredentials)
	b.start()
	b.Notify(func() {
		abort <- true
	})
	assert.True(t, <-abort, "should abort the build")
	b.Success()
}

func TestBuildOutputLimit(t *testing.T) {
	u := &updateTraceNetwork{}
	buildCredentials := &common.BuildCredentials{
		ID: successID,
	}
	b := newBuildTrace(u, buildOutputLimit, buildCredentials)
	b.start()

	// Write 500k to the buffer
	for i := 0; i < 100000; i++ {
		fmt.Fprint(b, "abcde")
	}
	b.Success()
	assert.True(t, len(*u.trace) < 2000, "the output should be less than 2000 bytes")
	assert.Contains(t, *u.trace, "Build log exceeded limit")
}

func TestBuildFinishRetry(t *testing.T) {
	traceFinishRetryInterval = time.Microsecond

	u := &updateTraceNetwork{}
	buildCredentials := &common.BuildCredentials{
		ID: retryID,
	}
	b := newBuildTrace(u, buildOutputLimit, buildCredentials)
	b.start()
	b.Success()
	assert.Equal(t, 5, u.count, "it should retry a few times")
	assert.Equal(t, common.Success, u.state)
}

func TestBuildForceSend(t *testing.T) {
	traceUpdateInterval = 0
	traceForceSendInterval = time.Minute

	u := &updateTraceNetwork{}
	buildCredentials := &common.BuildCredentials{
		ID: successID,
	}
	b := newBuildTrace(u, buildOutputLimit, buildCredentials)
	b.start()
	defer b.Success()

	fmt.Fprint(b, "test")

	started := time.Now()
	for time.Since(started) < time.Second {
		if u.trace != nil &&
			*u.trace == "test" {
			u.count = 0
			break
		}
	}
	assert.True(t, u.count == 0, "it didn't update the trace yet")
	assert.Equal(t, common.Running, u.state)

	traceForceSendInterval = 0

	started = time.Now()
	for time.Since(started) < time.Second {
		if u.count > 0 {
			break
		}
	}
	assert.True(t, u.count > 0, "it forcefully update trace more then once")
	assert.Equal(t, "test", *u.trace)
	assert.Equal(t, common.Running, u.state)
}
