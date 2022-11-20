package ospkg

import (
	"os/exec"
)

// ArchLinuxInstaller installs packages on Arch Linux.
type ArchLinuxInstaller struct{}

var _ Installer = (*ArchLinuxInstaller)(nil)

const pacmanCommand = "/usr/bin/pacman"

func (p *ArchLinuxInstaller) Install(pkg string) error {
	checkCmd := exec.Command(pacmanCommand, "-Qi", pkg)
	err := checkCmd.Run()
	if err == nil {
		return nil
	}

	installCmd := exec.Command(pacmanCommand, "-S", "--noconfirm", pkg)
	_, err = installCmd.CombinedOutput()
	return err
}
