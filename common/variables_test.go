package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVariableString(t *testing.T) {
	v := BuildVariable{"key", "value", false}
	assert.Equal(t, "key=value", v.String())
}

func TestPublicVariables(t *testing.T) {
	v1 := BuildVariable{"key", "value", false}
	v2 := BuildVariable{"public", "value", true}
	v3 := BuildVariable{"private", "value", false}
	all := BuildVariables{v1, v2, v3}
	public := all.Public()
	assert.Contains(t, public, v2)
	assert.NotContains(t, public, v1)
	assert.NotContains(t, public, v3)
}

func TestListVariables(t *testing.T) {
	v := BuildVariables{{"key", "value", false}}
	assert.Equal(t, []string{"key=value"}, v.StringList())
}

func TestGetVariable(t *testing.T) {
	v1 := BuildVariable{"key", "key_value", false}
	v2 := BuildVariable{"public", "public_value", true}
	v3 := BuildVariable{"private", "private_value", false}
	all := BuildVariables{v1, v2, v3}

	assert.Equal(t, "public_value", all.Get("public"))
	assert.Empty(t, all.Get("other"))
}

func TestParseVariable(t *testing.T) {
	v, err := ParseVariable("key=value=value2")
	assert.NoError(t, err)
	assert.Equal(t, BuildVariable{"key", "value=value2", false}, v)
}

func TestInvalidParseVariable(t *testing.T) {
	_, err := ParseVariable("some_other_key")
	assert.Error(t, err)
}

func TestVariablesExpansion(t *testing.T) {
	all := BuildVariables{
		{"key", "value_of_$public", false},
		{"public", "value_of_$undefined", true},
		{"private", "value_of_${public}", false},
	}

	expanded := all.Expand()
	assert.Len(t, expanded, 3)
	assert.Equal(t, expanded.Get("key"), "value_of_value_of_$undefined")
	assert.Equal(t, expanded.Get("public"), "value_of_")
	assert.Equal(t, expanded.Get("private"), "value_of_value_of_$undefined")
}
