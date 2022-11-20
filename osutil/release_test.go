package osutil

import (
	"testing"
)

func TestID(t *testing.T) {
	var tests = []struct {
		osReleaseFilePath, expected string
	}{
		{
			osReleaseFilePath: "testdata/os-release-rhel77",
			expected:          "rhel",
		},
		{
			osReleaseFilePath: "testdata/os-release-ubuntu2004",
			expected:          "ubuntu",
		},
	}

	for _, test := range tests {
		osRelease = test.osReleaseFilePath
		actual := ID()
		if test.expected != actual {
			t.Fatalf("Expected: %q, got :%q", test.expected, actual)
		}
	}
}
