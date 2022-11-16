package main

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/miekg/gitopper/gitcmd"
	"github.com/miekg/gitopper/log"
	"go.science.ru.nl/mountinfo"
)

// Service contains the service configuration tied to a specific machine.
type Service struct {
	Upstream string // The URL of the (upstream) Git repository.
	Service  string // Identifier for the service - will be used for action.
	Machine  string // Identifier for this machine - may be shared with multiple machines.
	Package  string // The package that might need installing.
	Action   string // The systemd action to take when files have changed.
	Mount    string // Together with Service this is the directory where the sparse git repo is checked out.
	Dirs     []Dir  // How to map our local directories to the git repository.

	State
	m *sync.RWMutex // protects state
}

type Dir struct {
	Local  string // The directory on the local filesystem.
	Link   string // The subdirectory inside the git repo to map to.
	Single bool   // unused... is a single file?
}

// Current State of a service.
type State int

const (
	StateOK State = iota
	StateFreeze
	StateRollback
)

// merge merges anything defined in s1 into s and returns the new Service. Currently this is only
// done for the Upstream field.
func (s Service) merge(s1 Service) Service {
	if s1.Upstream != "" {
		s.Upstream = s1.Upstream
	}
	return s
}

func (s Service) newGitCmd() *gitcmd.Git {
	dirs := []string{}
	for _, d := range s.Dirs {
		dirs = append(dirs, d.Link)
	}
	return gitcmd.New(s.Upstream, path.Join(s.Mount, s.Service), dirs)
}

// TrackUpstream does all the administration to track upstream and issue systemctl commands to keep the process
// informed.
func (s Service) trackUpstream(stop chan bool) {
	gc := s.newGitCmd()
	log.Infof("Launching tracking routine for %q/%q", s.Machine, s.Service)
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
			log.Infof("Machine %q, no diff in repo %q", s.Machine, s.Upstream)
			continue
		}

		log.Infof("Machine %q, diff in repo %q, pinging service: %s", s.Machine, s.Upstream, s.Service)
		if err := s.systemctl(); err != nil {
			// usually this tell you nothing, because atual error is only visible with journald
			log.Warningf("Machine %q, error running systemcl: %s", s.Machine, err)
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

		if ok, err := mountinfo.Mounted(d.Local); err == nil && ok {
			log.Infof("Directory %q is already mounted", d.Local)
			continue
		}

		ctx := context.TODO()
		cmd := exec.CommandContext(ctx, "mount", "-r", "--bind", gitdir, d.Local)
		log.Infof("running %v", cmd.Args)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to mount %q: %s", gitdir, err)
		}
	}
	return nil
}
