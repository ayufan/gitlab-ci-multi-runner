package helpers

import (
	"testing"
	"os"
)

func SkipIntegrationTest(t *testing.T) bool {
	if os.Getenv("INTEGRATION_TESTS") != "" {
		return false
	}
	t.Skip("Enable this tests with INTEGRATION_TESTS=1")
	return true
}
