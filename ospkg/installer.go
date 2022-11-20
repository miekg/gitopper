package ospkg

import "github.com/miekg/gitopper/osutil"

// Installer represents OS package installation tool.
type Installer interface {
	Install(pkg string) error // Install installs the given package at the given version.
}

// New returns an Installer suited for the current system, or the NoopInstaller when none are found.
func New() Installer {
	switch osutil.ID() {
	case "debian", "ubuntu":
		return new(DebianInstaller)
	case "arch":
		return new(ArchLinuxInstaller)
	}
	return new(NoopInstaller)
}
