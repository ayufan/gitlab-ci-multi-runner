package helpers

import (
	"testing"
	"os"
	"os/exec"
)

func SkipIntegrationTest(t *testing.T, apps... string) bool {
	if os.Getenv("INTEGRATION_TESTS") == "" {
		t.Skip("Enable this tests with INTEGRATION_TESTS=1")
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
