package main

const (
	// mountFmt is the bind mount command to be executed.
	mountFmt = "mount -r --bind %s %s" // olddir newdir

	// systemdFmt is the systemctl command to be executed.
	systemdFmt = "systemctl %s %s" // action service
)
