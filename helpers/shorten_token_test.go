package helpers

import (
	"testing"
)

func TestShortenToken(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{"short", "short"},
		{"veryverylongtoken", "veryvery"},
	}

	for _, test := range tests {
		actual := ShortenToken(test.in)
		if actual != test.out {
			t.Error("Expected ", test.out, ", get ", actual)
		}
	}
}
