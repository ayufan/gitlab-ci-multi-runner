package network

import (
	"bufio"
	"bytes"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"time"
)

type clientBuildTrace struct {
	*io.PipeWriter

	client *GitLabClient
	config common.RunnerConfig
	id     int
	limit  int64
	abort  func()

	log      bytes.Buffer
	state    common.BuildState
	finished chan bool

	sentTrace int
	sentTime  time.Time
	sentState common.BuildState
}

func (c *clientBuildTrace) Success() {
	if c.state != common.Running {
		return
	}
	c.state = common.Success
	c.finish()
}

func (c *clientBuildTrace) Fail(err error) {
	if c.state != common.Running {
		return
	}
	c.state = common.Failed
	c.finish()
}

func (c *clientBuildTrace) Notify(abort func()) {
	c.abort = abort
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
	c.finished <- true

	// Do final upload of build trace
	for {
		if c.update() != common.UpdateFailed {
			break
		} else {
			time.Sleep(common.UpdateRetryInterval * time.Second)
		}
	}
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
			c.log.WriteRune(r)
		} else {
			// ignore invalid characters
			continue
		}

		if c.log.Len() > limit {
			output := fmt.Sprintf("\n%sBuild log exceeded limit of %v bytes.%s\n",
				helpers.ANSI_BOLD_RED,
				limit,
				helpers.ANSI_RESET,
			)
			c.log.WriteString(output)
			stopped = true
		}
	}
}

func (c *clientBuildTrace) update() common.UpdateState {
	state := c.state
	trace := c.log.String()

	if c.sentState == state &&
		c.sentTrace == len(trace) &&
		time.Since(c.sentTime) < common.ForceTraceSentInterval {
		return common.UpdateSucceeded
	}

	upload := c.client.UpdateBuild(c.config, c.id, state, trace)
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
		case <-time.After(common.UpdateInterval):
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
