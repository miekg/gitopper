package ospkg

import (
	"os/exec"
)

// DebianInstaller installs packages on Debian/Ubuntu.
type DebianInstaller struct{}

var _ Installer = (*DebianInstaller)(nil)

const (
	aptGetCommand = "/usr/bin/apt-get"
	dpkgCommand   = "/usr/bin/dpkg"
)

func (p *DebianInstaller) Install(pkg string) error {
	checkCmd := exec.Command(dpkgCommand, "-s", pkg)
	if err := checkCmd.Run(); err == nil {
		return nil
	}

	args := []string{"-qq", "--assume-yes", "--no-install-recommends", "install", pkg}
	installCmd := exec.Command(aptGetCommand, args...)
	_, err := installCmd.CombinedOutput()
	return err
}
