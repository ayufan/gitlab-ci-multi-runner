package helpers

import (
	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTOMLOmitEmpty(t *testing.T) {
	var config struct {
		Value int `toml:"value,omitzero"`
	}

	// This test is intended to test this not fixed problem:
	// https://github.com/chowey/toml/commit/8249b7bc958927e7a8b392f66adbe4d5ead737d9
	text := `Value=10`
	_, err := toml.Decode(text, &config)
	require.NoError(t, err)
	assert.Equal(t, 10, config.Value)
}
