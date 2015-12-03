package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVariableString(t *testing.T) {
	v := BuildVariable{"key", "value", false, false}
	assert.Equal(t, "key=value", v.String())
}

func TestPublicAndInternalVariables(t *testing.T) {
	v1 := BuildVariable{"key", "value", false, false}
	v2 := BuildVariable{"public", "value", true, false}
	v3 := BuildVariable{"private", "value", false, true}
	all := BuildVariables{v1, v2, v3}
	public := all.PublicOrInternal()
	assert.NotContains(t, public, v1)
	assert.Contains(t, public, v2)
	assert.Contains(t, public, v3)
}

func TestListVariables(t *testing.T) {
	v := BuildVariables{{"key", "value", false, false}}
	assert.Equal(t, []string{"key=value"}, v.StringList())
}

func TestGetVariable(t *testing.T) {
	v1 := BuildVariable{"key", "key_value", false, false}
	v2 := BuildVariable{"public", "public_value", true, false}
	v3 := BuildVariable{"private", "private_value", false, false}
	all := BuildVariables{v1, v2, v3}

	assert.Equal(t, "public_value", all.Get("public"))
	assert.Empty(t, all.Get("other"))
}

func TestParseVariable(t *testing.T) {
	v, err := ParseVariable("key=value=value2")
	assert.NoError(t, err)
	assert.Equal(t, BuildVariable{"key", "value=value2", false, false}, v)
}

func TestInvalidParseVariable(t *testing.T) {
	_, err := ParseVariable("some_other_key")
	assert.Error(t, err)
}

func TestVariablesExpansion(t *testing.T) {
	all := BuildVariables{
		{"key", "value_of_$public", false, false},
		{"public", "some_value", true, false},
		{"private", "value_of_${public}", false, false},
		{"public", "value_of_$undefined", true, false},
	}

	expanded := all.Expand()
	assert.Len(t, expanded, 4)
	assert.Equal(t, expanded.Get("key"), "value_of_value_of_$undefined")
	assert.Equal(t, expanded.Get("public"), "value_of_")
	assert.Equal(t, expanded.Get("private"), "value_of_value_of_$undefined")
	assert.Equal(t, expanded.ExpandValue("${public} ${private}"), "value_of_ value_of_value_of_$undefined")
}
