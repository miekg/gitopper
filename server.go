package main

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"time"

	"github.com/miekg/gitopper/gitcmd"
	"github.com/miekg/gitopper/log"
)

// Service contains the service configuration tied to a specific machine.
type Service struct {
	Service  string // Identifier for the service - will be used for action.
	Machine  string // Identifier for this machine - may be shared with multiple machines.
	Package  string // The package that might need installing.
	Action   string `toml:"omitempty"` // The systemd action to take when files have changed.
	Upstream string `toml:"omitempty"` // The URL of the (upstream) Git repository.
	Mount    string // Together with Service this is the directory where the sparse git repo is checked out.
	Dirs     []Dir  // How to map our local directories to the git repository.
}

type Dir struct {
	Local  string // The directory on the local filesystem.
	Link   string // The subdirectory inside the git repo to map to.
	Single bool   // unused... is a single file?
}

// merge merges anything defined in upper into lower and returns the new Machine. Currently this is only
// done for the Upstream field.
func merge(upper, lower Service) Service {
	if upper.Upstream != "" {
		lower.Upstream = upper.Upstream
	}
	return lower
}

func NewGitCmd(s Service) *gitcmd.Git {
	dirs := []string{}
	for _, d := range s.Dirs {
		dirs = append(dirs, d.Link)
	}
	return gitcmd.New(s.Upstream, path.Join(s.Mount, s.Service), dirs)
}

// TrackUpstream does all the administration to track upstream and issue systemctl commands to keep the process
// informed.
func (s Service) TrackUpstream(stop chan bool) {
	gc := NewGitCmd(s)
	for {
		time.Sleep(30 * time.Second) // track in service?
		if err := gc.Pull(); err != nil {
			log.Warningf("Machine %q, error pulling repo %q: %s", s.Machine, s.Upstream, err)
			// TODO: metric pull errors, pull ok, pull latency??
			continue
		}

		changed, err := gc.Diff()
		if err != nil {
			log.Warningf("Machine %q, error diffing repo %q: %s", s.Machine, s.Upstream, err)
		}
		if !changed {
			continue
		}

	}
}

func (s Service) systemctl() error {
	ctx := context.TODO()
	cmd := exec.CommandContext(ctx, "systemctl", s.Action, s.Service)
	fmt.Printf("%+v\n", cmd)
	return nil
}

func (s Service) bindmount() error {
	for _, d := range s.Dirs {
		gitdir := path.Join(s.Mount, s.Service)
		gitdir = path.Join(gitdir, d.Link)

		ctx := context.TODO()
		cmd := exec.CommandContext(ctx, "mount", "-r", "--bind", d.Link, gitdir)
		fmt.Printf("%+v\n", cmd)
	}
	return nil
}
