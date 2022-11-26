package ospkg

import (
	"os/exec"

	"go.science.ru.nl/log"
)

// ArchLinuxInstaller installs packages on Arch Linux.
type ArchLinuxInstaller struct{}

var _ Installer = (*ArchLinuxInstaller)(nil)

const pacmanCommand = "/usr/bin/pacman"

func (p *ArchLinuxInstaller) Install(pkg string) error {
	installCmd := exec.Command(pacmanCommand, "-S", "--noconfirm", pkg)
	out, err := installCmd.CombinedOutput()
	if err != nil {
		log.Warningf("Install failed: %s", out)
	} else {
		log.Infof("Already installed or re-installed %q", pkg)
	}
	return err
}
