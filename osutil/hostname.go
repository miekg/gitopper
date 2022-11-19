package osutil

import "os"

// Hostname returns the hostname of the current machine.
func Hostname() string {
	h, _ := os.Hostname()
	return h
}
