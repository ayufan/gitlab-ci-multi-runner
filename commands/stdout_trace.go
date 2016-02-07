package commands

import "os"

type stdoutTrace struct {
}

func (s *stdoutTrace) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

func (s *stdoutTrace) Success() {
}

func (s *stdoutTrace) Fail(err error) {
}

func (s *stdoutTrace) Notify(abort func()) {
}
