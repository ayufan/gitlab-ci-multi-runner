package helpers

import (
	"os/exec"
	"strconv"
)

func SetProcessGroup(cmd *exec.Cmd) {
}

func KillProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	cmd.Process.Kill()
}
