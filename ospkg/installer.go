package ospkg

import (
	"github.com/miekg/gitopper/osutil"
	"go.science.ru.nl/log"
)

// Installer represents OS package installation tool.
type Installer interface {
	Install(pkg string) error
}

// New returns an Installer suited for the current system, or the NoopInstaller when none are found.
func New() Installer {
	switch osutil.ID() {
	case "debian", "ubuntu":
		return new(DebianInstaller)
	case "arch":
		return new(ArchLinuxInstaller)
	}
	log.Warningf("Returning Noop package installer for %s", osutil.ID())
	return new(NoopInstaller)
}
