package helpers

import (
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
		if actual != test.out {
			t.Error("Expected ", test.out, ", get ", actual)
		}
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
