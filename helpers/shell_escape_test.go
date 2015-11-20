package helpers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestShellEscape(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{"standard string", "$'standard string'"},
		{"+\t\n\r&", "$'+\\t\\n\\r&'"},
		{"", "''"},
	}

	for _, test := range tests {
		actual := ShellEscape(test.in)
		assert.Equal(t, test.out, actual, "src=%v", test.in)
	}
}

func TestToBackslash(t *testing.T) {

	result := ToBackslash("smb://user/me/directory")
	expected := "smb:\\\\user\\me\\directory"

	if result != expected {
		t.Error("Expected", expected, ", got ", result)
	}
}

func TestToSlash(t *testing.T) {

	result := ToSlash("smb:\\\\user\\me\\directory")
	expected := "smb://user/me/directory"

	if result != expected {
		t.Error("Expected", expected, ", got ", result)
	}
}
