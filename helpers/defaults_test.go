package helpers

import (
	"testing"
)

func TestIsEmpty(t *testing.T) {
	var nilString string
	var tests = []struct {
		in  string
		out bool
	}{
		{nilString, true},
		{"", true},
		{"notempty", false},
	}

	for _, test := range tests {
		actual := IsEmpty(&test.in)
		if actual != test.out {
			t.Error("Expected ", test.out, ", get ", actual)
		}
	}
}

func TestStringOrDefault(t *testing.T) {
	var nilString string
	var tests = []struct {
		in  string
		in2 string
		out string
	}{
		{nilString, "default", "default"},
		{"", "default", "default"},
		{"notempty", "default", "notempty"},
	}

	for _, test := range tests {
		actual := StringOrDefault(&test.in, test.in2)
		if actual != test.out {
			t.Error("Expected ", test.out, ", get ", actual)
		}
	}
}

func TestNonZeroOrDefault(t *testing.T) {
	var nilInt int
	var tests = []struct {
		in  int
		in2 int
		out int
	}{
		{2, 42, 2},
		{nilInt, 42, 42},
		{0, 42, 42},
		{-5, 42, 42},
	}

	for _, test := range tests {
		actual := NonZeroOrDefault(&test.in, test.in2)
		if actual != test.out {
			t.Error("Expected ", test.out, ", get ", actual)
		}
	}
}

func TestBoolOrDefault(t *testing.T) {
	var nilBool bool
	var tests = []struct {
		in  bool
		in2 bool
		out bool
	}{
		{nilBool, false, false},
		{false, true, false},
		{true, false, true},
	}

	for _, test := range tests {
		actual := BoolOrDefault(&test.in, test.in2)
		if actual != test.out {
			t.Error("Expected ", test.out, ", get ", actual)
		}
	}
}
