package network

import (
	"bufio"
	"bytes"
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

type clientBuildTrace struct {
	*io.PipeWriter

	client common.Network
	config common.RunnerConfig
	id     int
	limit  int64
	abort  func()

	log      bytes.Buffer
	lock     sync.RWMutex
	state    common.BuildState
	finished chan bool

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
	go c.process(reader)
	go c.watch()
}

func (c *clientBuildTrace) finish() {
	c.Close()
	c.finished <- true

	// Do final upload of build trace
	for {
		if c.update() != common.UpdateFailed {
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
	c.lock.RLock()
	state := c.state
	tracePart := c.log.String()
	c.lock.RUnlock()

	if c.sentState == state &&
		time.Since(c.sentTime) < traceForceSendInterval {
		return common.UpdateSucceeded
	}

	traceUpdate := c.client.SendTracePart(c.config, c.id, tracePart)
	if traceUpdate == common.UpdateSucceeded {
		c.lock.Lock()
		c.log.Reset()
		c.lock.Unlock()
	}

	stateUpdate := c.client.UpdateBuildState(c.config, c.id, state)
	if stateUpdate == common.UpdateSucceeded {
		c.sentState = state
		c.sentTime = time.Now()
	}
	return stateUpdate
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

func newBuildTrace(client common.Network, config common.RunnerConfig, id int) *clientBuildTrace {
	return &clientBuildTrace{
		client: client,
		config: config,
		id:     id,
	}
}
