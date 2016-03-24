package common

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
)

type Logging struct {
	LogEntry *logrus.Entry
	LogTrace BuildTrace
}

func (e *Logging) Debugln(args ...interface{}) {
	if e.LogEntry != nil {
		e.LogEntry.Debugln(args...)
	}
}

func (e *Logging) Println(args ...interface{}) {
	if e.LogTrace != nil {
		fmt.Fprintln(e.LogTrace, args...)

		if e.LogTrace.IsStdout() {
			return
		}
	}

	if len(args) == 0 {
		return
	}

	if e.LogEntry != nil {
		e.LogEntry.Println(args...)
	}
}

func (e *Logging) Infoln(args ...interface{}) {
	if e.LogTrace != nil {
		fmt.Fprint(e.LogTrace, helpers.ANSI_BOLD_GREEN+fmt.Sprintln(args...)+helpers.ANSI_RESET)

		if e.LogTrace.IsStdout() {
			return
		}
	}

	if len(args) == 0 {
		return
	}

	if e.LogEntry != nil {
		e.LogEntry.Println(args...)
	}
}

func (e *Logging) Warningln(args ...interface{}) {
	if e.LogTrace != nil {
		fmt.Fprint(e.LogTrace, helpers.ANSI_BOLD_YELLOW+"WARNING: "+fmt.Sprintln(args...)+helpers.ANSI_RESET)

		if e.LogTrace.IsStdout() {
			return
		}
	}

	if len(args) == 0 {
		return
	}

	if e.LogEntry != nil {
		e.LogEntry.Warningln(args...)
	}
}

func (e *Logging) Errorln(args ...interface{}) {
	if e.LogTrace != nil {
		fmt.Fprint(e.LogTrace, helpers.ANSI_BOLD_RED+"ERROR: "+fmt.Sprintln(args...)+helpers.ANSI_RESET)

		if e.LogTrace.IsStdout() {
			return
		}
	}

	if e.LogEntry != nil {
		e.LogEntry.Errorln(args...)
	}
}
