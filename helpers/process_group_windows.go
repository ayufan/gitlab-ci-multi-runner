package helpers

import (
	"os/exec"
)

func SetProcessGroup(cmd *exec.Cmd) {
}

func KillProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	cmd.Process.Kill()
}
