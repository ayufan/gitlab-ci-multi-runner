// +build darwin dragonfly freebsd linux netbsd openbsd

package helpers

import "github.com/ramr/go-reaper"

func Reap() {
	reaper.Reap()
}
