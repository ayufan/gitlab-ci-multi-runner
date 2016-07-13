package common

import (
	"io"
)

type Trace struct {
	Writer io.Writer
}

func (s *Trace) Write(p []byte) (n int, err error) {
	return s.Writer.Write(p)
}

func (s *Trace) Success() {
}

func (s *Trace) Fail(err error) {
}

func (s *Trace) Notify(abort func()) {
}

func (s *Trace) IsStdout() bool {
	return true
}
