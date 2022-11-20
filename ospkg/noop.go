package ospkg

// NoopManager implements a no-op of the Installer interface.
// Its purpose is to enable scenarios where no package handling is required,
// i.e. the necessary executables are already available on the host.
type NoopInstaller struct{}

var _ Installer = (*NoopInstaller)(nil)

func (p *NoopInstaller) Install(pkg string) error { return nil }
