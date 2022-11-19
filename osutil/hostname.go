package osutil

import "os"

// Hostname returns the hostname of this host. If it fails the value $HOSTNAME is returned.
func Hostname() string {
	h, err := os.Hostname()
	if err != nil {
		h = os.Getenv("HOSTNAME")
	}
	return h
}
