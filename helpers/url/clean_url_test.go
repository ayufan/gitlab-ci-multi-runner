package url_helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemovingAllSensitiveData(t *testing.T) {
	url := CleanURL("https://user:password@gitlab.com/gitlab?key=value#fragment")
	assert.Equal(t, "https://gitlab.com/gitlab", url)
}

func TestInvalidURL(t *testing.T) {
	assert.Empty(t, CleanURL("://invalid URL"))
}
