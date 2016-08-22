package archives

import (
	"os"
	"sync"
)

// When extracting an archive, the same PathError.Op may be repeated for every
// file in the archive; use pathErrorTracker to suppress repetitious log output
type pathErrorTracker struct {
	sync.Mutex
	seenOps map[string]bool
}

// check whether the error is actionable, which is to say, not nil and either
// not a PathError, or a novel PathError
func (p *pathErrorTracker) actionable(e error) bool {
	pathErr, isPathErr := e.(*os.PathError)
	if e == nil || isPathErr && pathErr == nil {
		return false
	}

	if !isPathErr {
		return true
	}

	p.Lock()
	defer p.Unlock()

	seen := p.seenOps[pathErr.Op]
	p.seenOps[pathErr.Op] = true

	// actionable if *not* seen before
	return !seen
}

func newPathErrorTracker() *pathErrorTracker {
	return &pathErrorTracker{
		seenOps: make(map[string]bool),
	}
}
