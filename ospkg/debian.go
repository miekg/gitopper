package ospkg

import (
	"os/exec"

	"go.science.ru.nl/log"
)

// DebianInstaller installs packages on Debian/Ubuntu.
type DebianInstaller struct{}

var _ Installer = (*DebianInstaller)(nil)

const aptGetCommand = "/usr/bin/apt-get"

func (p *DebianInstaller) Install(pkg string) error {
	args := []string{"-qq", "--assume-yes", "--no-install-recommends", "install", pkg}
	installCmd := exec.Command(aptGetCommand, args...)
	out, err := installCmd.CombinedOutput()
	if err != nil {
		log.Warningf("Install failed: %s", out)
	} else {
		log.Infof("Already installed or re-installed %q", pkg)
	}
	return err
}
