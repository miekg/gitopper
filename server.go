package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/miekg/gitopper/gitcmd"
	"github.com/miekg/gitopper/osutil"
	"go.science.ru.nl/log"
	"go.science.ru.nl/mountinfo"
)

// Service contains the service configuration tied to a specific machine.
type Service struct {
	Upstream string        // The URL of the (upstream) Git repository.
	Branch   string        // The branch to track (defaults to 'main').
	Service  string        // Identifier for the service - will be used for action.
	Machine  string        // Identifier for this machine - may be shared with multiple machines.
	Package  string        // The package that might need installing.
	User     string        // what user to use for checking out the repo.
	Action   string        // The systemd action to take when files have changed.
	Mount    string        // Together with Service this is the directory where the sparse git repo is checked out.
	Dirs     []Dir         // How to map our local directories to the git repository.
	Duration time.Duration `toml:"_"` // how much to sleep between pulls

	state        State
	stateInfo    string    // Extra info some states carry.
	stateStamp   time.Time // When did state change (UTC).
	hash         string    // Git hash of the current git checkout.
	sync.RWMutex           // Protects state and friends.
}

type Dir struct {
	Local string // The directory on the local filesystem.
	Link  string // The subdirectory inside the git repo to map to.
}

// Current State of a service.
type State int

const (
	StateOK       State = iota // The service is running as it should.
	StateFreeze                // The service is locked to the current commit, no further updates are done.
	StateRollback              // The service is rolled back and locked to that commit, no further updates are done.
	StateBroken                // The service is broken, i.e. didn't start, systemctl error, etc.
)

func (s State) String() string {
	switch s {
	case StateOK:
		return "OK"
	case StateFreeze:
		return "FREEZE"
	case StateRollback:
		return "ROLLBACK"
	case StateBroken:
		return "BROKEN"
	}
	return ""
}

func (s *Service) State() (State, string) {
	s.RLock()
	defer s.RUnlock()
	return s.state, s.stateInfo
}

func (s *Service) SetState(st State, info string) {
	s.Lock()
	defer s.Unlock()
	s.stateStamp = time.Now().UTC()
	s.state = st
	s.stateInfo = info

	metricServiceHash.WithLabelValues(s.Service, s.hash, s.state.String()).Set(1)
}

func (s *Service) Hash() string {
	s.RLock()
	defer s.RUnlock()
	return s.hash
}

func (s *Service) SetHash(h string) {
	s.Lock()
	defer s.Unlock()
	s.hash = h
}

func (s *Service) Change() time.Time {
	s.RLock()
	defer s.RUnlock()
	return s.stateStamp
}

// merge merges anything defined in s1 into s and returns the new Service. Currently this is only
// done for the Upstream field.
func (s *Service) merge(s1 *Service, d time.Duration) *Service {
	if s1.Upstream != "" {
		s.Upstream = s1.Upstream
	}
	s.Duration = d
	if s.Branch == "" {
		s.Branch = "main"
	}
	return s
}

// forMe compares the hostnames with the service machine name, it there is a match for service is for us.
func (s *Service) forMe(hostnames []string) bool {
	for _, h := range hostnames {
		if h == s.Machine {
			return true
		}
	}
	return false
}

func (s *Service) newGitCmd() *gitcmd.Git {
	dirs := []string{}
	for _, d := range s.Dirs {
		dirs = append(dirs, d.Link)
	}
	return gitcmd.New(s.Upstream, s.Branch, path.Join(s.Mount, s.Service), s.User, dirs)
}

// TrackUpstream does all the administration to track upstream and issue systemctl commands to keep the process
// informed.
func (s *Service) trackUpstream(ctx context.Context) {
	gc := s.newGitCmd()

	log.Infof("Launched tracking routine for %q/%q", s.Machine, s.Service)

	for {
		s.SetHash(gc.Hash())
		state, info := s.State()

		select {
		case <-time.After(s.Duration):
		case <-ctx.Done():
			return
		}

		// this in now only done once... because we set state to broken... Should we keep trying??
		if state == StateRollback && info != s.hash {
			if err := gc.Rollback(info); err != nil {
				log.Warningf("Machine %q, error rollback repo %q to %q: %s", s.Machine, s.Upstream, info, err)
				s.SetState(StateBroken, fmt.Sprintf("error rolling back %q to %q: %s", s.Upstream, info, err))
				continue
			}

			if err := s.systemctl(); err != nil {
				log.Warningf("Machine %q, error running systemctl: %s", s.Machine, err)
				s.SetState(StateBroken, fmt.Sprintf("error running systemctl %q: %s", s.Upstream, err))
				continue
			}
			log.Warningf("Machine %q, successfully rollback repo %q to %s", s.Machine, s.Upstream, info)
			s.SetState(StateFreeze, "ROLLBACK: "+info)
			continue
		}

		if state, _ := s.State(); state == StateFreeze || state == StateRollback {
			log.Warningf("Machine %q is service %q is %s, not pulling", s.Machine, s.Service, state)
			continue
		}

		changed, err := gc.Pull()
		if err != nil {
			log.Warningf("Machine %q, error pulling repo %q: %s", s.Machine, s.Upstream, err)
			s.SetState(StateBroken, fmt.Sprintf("error pulling %q: %s", s.Upstream, err))
			continue
		}

		if !changed {
			log.Infof("Machine %q, no diff in repo %q", s.Machine, s.Upstream)
			continue
		}

		s.SetHash(gc.Hash())
		state, info = s.State()
		s.SetState(state, info)

		log.Infof("Machine %q, diff in repo %q, pinging service: %s", s.Machine, s.Upstream, s.Service)
		if err := s.systemctl(); err != nil {
			log.Warningf("Machine %q, error running systemctl: %s", s.Machine, err)
			s.SetState(StateBroken, fmt.Sprintf("error running systemctl %q: %s", s.Upstream, err))
			continue
		}
	}
}

func (s *Service) systemctl() error {
	if s.Action == "" {
		return nil
	}
	ctx := context.TODO()
	cmd := exec.CommandContext(ctx, "systemctl", s.Action, s.Service)
	log.Infof("running %v", cmd.Args)
	return cmd.Run()
}

// bindmount sets up the bind mount, the return integer returns how many mounts were performed.
func (s *Service) bindmount() (int, error) {
	mounted := 0
	for _, d := range s.Dirs {
		if d.Local == "" {
			continue
		}

		gitdir := path.Join(s.Mount, s.Service)
		gitdir = path.Join(gitdir, d.Link)

		if !exists(d.Local) {
			if err := os.MkdirAll(d.Local, 0775); err != nil {
				log.Errorf("Directory %q can not be created", d.Local)
				return 0, fmt.Errorf("failed to create directory %q: %s", d.Local, err)
			}
			// set base to correct owner
			uid, gid := osutil.User(s.User)
			if err := os.Chown(path.Base(d.Local), int(uid), int(gid)); err != nil {
				log.Errorf("Directory %q can not be chown to %q: %s", d.Local, s.User, err)
				return 0, fmt.Errorf("failed to chown directory %q to %q: %s", d.Local, s.User, err)
			}
		}

		if ok, err := mountinfo.Mounted(d.Local); err == nil && ok {
			log.Infof("Directory %q is already mounted", d.Local)
			continue
		}

		ctx := context.TODO()
		cmd := exec.CommandContext(ctx, "mount", "-r", "--bind", gitdir, d.Local)
		log.Infof("running %v", cmd.Args)
		err := cmd.Run()
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if e := exitError.ExitCode(); e != 0 {
					return 0, fmt.Errorf("failed to mount %q, exit code %d", gitdir, e)
				}
			}
			return 0, fmt.Errorf("failed to mount %q: %s", gitdir, err)
		}
		mounted++

	}
	return mounted, nil
}

func selfService(upstream, branch, mount, dir string) *Service {
	if upstream == "" || branch == "" || mount == "" || dir == "" {
		return nil
	}
	return &Service{
		Upstream: upstream,
		Branch:   branch,
		Mount:    mount,
		Action:   "", // empty, because with -r true systemd will just restart us
		Service:  "gitopper",
		Machine:  osutil.Hostname(),
		Dirs:     []Dir{{Link: dir}},
	}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
