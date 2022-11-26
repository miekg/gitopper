package osutil

import (
	"bytes"
	"os"
)

var (
	// this is a variable so it can be overridden during unit-testing.
	osRelease = "/etc/os-release"
)

// ID returns the ID of the system as specific in the osRelease file.
func ID() string {
	buf, err := os.ReadFile(osRelease)
	if err != nil {
		return ""
	}
	i := bytes.Index(buf, []byte("\nID=")) // want ^ID=
	if i == 0 {
		return ""
	}
	id := buf[i+len("\nID="):]
	j := bytes.Index(id, []byte("\n"))
	if j == 0 {
		return ""
	}
	// Some attributes are quoted, some are not. Cover both.
	id = bytes.ReplaceAll(id[:j], []byte("\""), []byte{})
	return string(id)
}
