package common

import (
	"io"
	"os"
)

type Trace struct {
	Writer io.Writer
	Abort  func()
}

func (s *Trace) Write(p []byte) (n int, err error) {
	if s.Writer == nil {
		return 0, os.ErrInvalid
	}
	return s.Writer.Write(p)
}

func (s *Trace) Success() {
}

func (s *Trace) Fail(err error) {
}

func (s *Trace) Notify(abort func()) {
	s.Abort = abort
}

func (s *Trace) IsStdout() bool {
	return true
}
