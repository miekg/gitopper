package main

// Machine contains the configuration for a specific machine.
type Machine struct {
	Name    string // Unique identfier for this machine.
	Package string // The package that might need installing.
	Action  string `toml:"omitempty"` // The systemd action to take when files have changed.
	URL     string // The URL of the (remote) Git repository.
	Mount   string // Directory where the sparse git repo is checked out.
	Dirs    []Dir  // How to map our local directories to the git repository.
}

type Dir struct {
	Local  string // The directory on the local filesystem.
	Link   string // The subdirectory inside the git repo to map to.
	Single bool   // unused... is a single file?
}

// merge merges anything defined in upper into lower and returns the new Machine. Currently this is only
// done for the URL field.
func merge(upper, lower Machine) Machine {
	if upper.URL != "" {
		lower.URL = upper.URL
	}
	return lower
}
