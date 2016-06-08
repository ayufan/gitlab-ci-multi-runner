package network

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"sync"
	"time"
)

var traceUpdateInterval = common.UpdateInterval
var traceForceSendInterval = common.ForceTraceSentInterval
var traceFinishRetryInterval = common.UpdateRetryInterval

type tracePatch struct {
	trace  bytes.Buffer
	offset int
	limit  int
}

func (tp *tracePatch) Patch() []byte {
	return tp.trace.Bytes()[tp.offset:tp.limit]
}

func (tp *tracePatch) Offset() int {
	return tp.offset
}

func (tp *tracePatch) Limit() int {
	return tp.limit
}

func (tp *tracePatch) SetNewOffset(newOffset int) {
	tp.offset = newOffset
}

func (tp *tracePatch) validateRange() bool {
	if tp.limit >= tp.offset {
		return true
	}

	return false
}

func newTracePatch(trace bytes.Buffer, offset int) (*tracePatch, error) {
	patch := &tracePatch{
		trace:  trace,
		offset: offset,
		limit:  trace.Len(),
	}

	if !patch.validateRange() {
		return nil, errors.New("Range is invalid, limit can't be less than offset")
	}

	return patch, nil
}

type clientBuildTrace struct {
	*io.PipeWriter

	client           common.Network
	config           common.RunnerConfig
	buildCredentials *common.BuildCredentials
	id               int
	limit            int64
	abort            func()

	incrementalAvailable bool

	log      bytes.Buffer
	lock     sync.RWMutex
	state    common.BuildState
	finished chan bool

	sentTrace int
	sentTime  time.Time
	sentState common.BuildState
}

func (c *clientBuildTrace) Success() {
	c.Fail(nil)
}

func (c *clientBuildTrace) Fail(err error) {
	c.lock.Lock()
	if c.state != common.Running {
		c.lock.Unlock()
		return
	}
	if err == nil {
		c.state = common.Success
	} else {
		c.state = common.Failed
	}
	c.lock.Unlock()

	c.finish()
}

func (c *clientBuildTrace) Notify(abort func()) {
	c.abort = abort
}

func (c *clientBuildTrace) IsStdout() bool {
	return false
}

func (c *clientBuildTrace) start() {
	reader, writer := io.Pipe()
	c.PipeWriter = writer
	c.finished = make(chan bool)
	c.state = common.Running
	c.incrementalAvailable = true
	go c.process(reader)
	go c.watch()
}

func (c *clientBuildTrace) finish() {
	c.Close()
	c.finished <- true

	// Do final upload of build trace
	for {
		if c.staleUpdate() != common.UpdateFailed {
			return
		}
		time.Sleep(traceFinishRetryInterval)
	}
}

func (c *clientBuildTrace) writeRune(r rune, limit int) (n int, err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	n, err = c.log.WriteRune(r)
	if c.log.Len() < limit {
		return
	}

	output := fmt.Sprintf("\n%sBuild log exceeded limit of %v bytes.%s\n",
		helpers.ANSI_BOLD_RED,
		limit,
		helpers.ANSI_RESET,
	)
	c.log.WriteString(output)
	err = io.EOF
	return
}

func (c *clientBuildTrace) process(pipe *io.PipeReader) {
	defer pipe.Close()

	stopped := false
	limit := c.config.OutputLimit
	if limit == 0 {
		limit = common.DefaultOutputLimit
	}
	limit *= 1024

	reader := bufio.NewReader(pipe)
	for {
		r, s, err := reader.ReadRune()
		if s <= 0 {
			break
		} else if stopped {
			// ignore symbols if build log exceeded limit
			continue
		} else if err == nil {
			_, err = c.writeRune(r, limit)
			if err == io.EOF {
				stopped = true
			}
		} else {
			// ignore invalid characters
			continue
		}
	}
}

func (c *clientBuildTrace) update() common.UpdateState {
	var update common.UpdateState

	if c.incrementalAvailable != false {
		update = c.incrementalUpdate()

		if update == common.UpdateNotFound {
			c.incrementalAvailable = false
			c.config.Log().Warningln("Incremental build update not available. Switching back to full build update")
		}
	}

	if c.incrementalAvailable == false {
		update = c.staleUpdate()
	}

	return update
}

func (c *clientBuildTrace) incrementalUpdate() common.UpdateState {
	c.lock.RLock()
	state := c.state
	trace := c.log
	c.lock.RUnlock()

	if c.sentState == state &&
		c.sentTrace == trace.Len() &&
		time.Since(c.sentTime) < traceForceSendInterval {
		return common.UpdateSucceeded
	}

	if c.sentState != state {
		c.client.UpdateBuild(c.config, c.id, state, nil)
		c.sentState = state
	}

	tracePatch, err := newTracePatch(trace, c.sentTrace)
	if err != nil {
		c.config.Log().Errorln("Error while creating a tracePatch", err.Error())
	}

	update := c.client.PatchTrace(c.config, c.buildCredentials, tracePatch)
	if update == common.UpdateNotFound {
		return update
	}

	if update == common.UpdateRangeMissmatch {
		update = c.resendPatch(c.buildCredentials.ID, c.config, c.buildCredentials, tracePatch)
	}

	if update == common.UpdateSucceeded {
		c.sentTrace = tracePatch.Limit()
		c.sentTime = time.Now()
	}

	return update
}

func (c *clientBuildTrace) resendPatch(id int, config common.RunnerConfig, buildCredentials *common.BuildCredentials, tracePatch common.BuildTracePatch) (update common.UpdateState) {
	config.Log().Warningln(id, "Resending trace patch due to range missmatch")

	update = c.client.PatchTrace(config, buildCredentials, tracePatch)
	if update == common.UpdateRangeMissmatch {
		config.Log().Errorln(id, "Appending trace to coordinator...", "failed due to range missmatch")
		update = common.UpdateFailed
	}

	return
}

func (c *clientBuildTrace) staleUpdate() common.UpdateState {
	c.lock.RLock()
	state := c.state
	trace := c.log.String()
	c.lock.RUnlock()

	if c.sentState == state &&
		c.sentTrace == len(trace) &&
		time.Since(c.sentTime) < traceForceSendInterval {
		return common.UpdateSucceeded
	}

	upload := c.client.UpdateBuild(c.config, c.id, state, &trace)
	if upload == common.UpdateSucceeded {
		c.sentTrace = len(trace)
		c.sentState = state
		c.sentTime = time.Now()
	}

	return upload
}

func (c *clientBuildTrace) watch() {
	for {
		select {
		case <-time.After(traceUpdateInterval):
			state := c.update()
			if state == common.UpdateAbort && c.abort != nil {
				c.abort()
				<-c.finished
				return
			}
			break

		case <-c.finished:
			return
		}
	}
}

func newBuildTrace(client common.Network, config common.RunnerConfig, buildCredentials *common.BuildCredentials) *clientBuildTrace {
	return &clientBuildTrace{
		client:           client,
		config:           config,
		buildCredentials: buildCredentials,
		id:               buildCredentials.ID,
	}
}
