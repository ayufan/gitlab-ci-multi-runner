package helpers

import (
	"os/exec"
	"testing"
)

func SkipIntegrationTests(t *testing.T, apps ...string) bool {
	if testing.Short() {
		t.Skip("Skipping long tests")
		return true
	}

	for _, app := range apps {
		_, err := exec.LookPath(app)
		if err != nil {
			t.Skip("Missing", app)
			return true
		}
	}

	return false
}
