// +build darwin dragonfly freebsd linux netbsd openbsd

package cli_helpers

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/Sirupsen/logrus"
)

func watchForGoroutinesDump() {
	// On USR1 dump stacks of all go routines
	dumpStacks := make(chan os.Signal, 1)
	signal.Notify(dumpStacks, syscall.SIGUSR1)
	for _ = range dumpStacks {
		buf := make([]byte, 1<<20)
		runtime.Stack(buf, true)
		logrus.Printf("=== received SIGUSR1 ===\n*** goroutine dump...\n%s\n*** end\n", buf)
	}
}
