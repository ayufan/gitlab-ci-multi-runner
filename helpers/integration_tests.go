package helpers

import (
	"os/exec"
	"testing"
)

func SkipIntegrationTests(t *testing.T, app ...string) bool {
	if testing.Short() {
		t.Skip("Skipping long tests")
		return true
	}

	if len(app) > 0 {
		cmd := exec.Command(app[0], app[1:]...)
		err := cmd.Run()
		if err != nil {
			t.Skip(app[0], "failed", err)
			return true
		}
	}
	return false
}
