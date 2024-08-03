package main

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/miekg/gitopper/gitcmd"
	"github.com/miekg/gitopper/osutil"
	"go.science.ru.nl/log"
	"go.science.ru.nl/mountinfo"
)

// Service contains the service configuration tied to a specific machine.
type Service struct {
	Upstream string // The URL of the (upstream) Git repository.
	Branch   string // The branch to track (defaults to 'main').
	Service  string // Identifier for the service - will be used for action.
	Machine  string // Identifier for this machine - may be shared with multiple machines.
	Package  string // The package that might need installing.
	User     string // what user to use for checking out the repo.
	Action   string // The systemd action to take when files have changed.
	Mount    string // Concatenated with server.Service this will be the directory where the git repo is checked out.
	Dirs     []Dir  // How to map our local directories to the git repository.

	pullNow chan struct{} // do an on demand pull

	mu         sync.RWMutex
	state      State
	stateInfo  string    // Extra info some states carry.
	stateStamp time.Time // When did state change (UTC).
	hash       string    // Git hash of the current git checkout.
}

type Dir struct {
	Local string // The directory on the local filesystem.
	Link  string // The subdirectory inside the git repo to map to.
	File  bool   // If true Local and Link are considered files.
}

// Current State of a service.
type State int

const (
	StateOK       State = iota // The service is running as it should.
	StateFreeze                // The service is locked to the current commit, no further updates are done.
	StateRollback              // The service is rolled back and locked to that commit, no further updates are done.
	StateBroken                // The service is broken, i.e. didn't start, systemctl error, etc.
	StateDiff                  // The service's git repo can't be reconciled with upstream for some reason.
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
	case StateDiff:
		return "DIFF"
	}
	return ""
}

func (s *Service) State() (State, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state, s.stateInfo
}

func (s *Service) SetState(st State, info string) {
	log.Infof("Service %q, setting to state: %s:s", s.Service, st)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stateStamp = time.Now().UTC()
	s.state = st
	s.stateInfo = info

	metricServiceState.WithLabelValues(s.Service).Set(float64(s.state))
	metricServiceTimestamp.WithLabelValues(s.Service).Set(float64(s.stateStamp.Unix()))
}

func (s *Service) Hash() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hash
}

func (s *Service) SetHash(h string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hash = h
}

func (s *Service) Change() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stateStamp
}

func (s *Service) signalPullNow() {
	s.pullNow <- struct{}{}
}

// merge merges anything defined in global into s when s doesn't specify it and returns the new Service.
func (s *Service) merge(global Global) *Service {
	if s.Upstream == "" {
		s.Upstream = global.Upstream
	}
	if s.Mount == "" {
		s.Mount = global.Mount
	}
	if s.Branch == "" {
		s.Branch = global.Branch
	}
	if s.Branch == "" {
		s.Branch = "main"
	}
	// TODO: Examine whether replacing pullNow needs to occur with synchronization due to reads.
	s.pullNow = make(chan struct{}) // TODO(miek): newService would be a better place for time.
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
func (s *Service) trackUpstream(ctx context.Context, duration time.Duration) {
	gc := s.newGitCmd()

	log.Infof("Launched tracking routine for %q", s.Service)
	s.SetHash(gc.Hash())
	s.SetBoot()
	state, info := s.State()
	s.SetState(state, info)

	for {
		s.SetHash(gc.Hash())

		select {
		case <-time.After(jitter(duration)):
		case <-s.pullNow:
		case <-ctx.Done():
			return
		}

		// this in now only done once... because we set state to broken... Should we keep trying??
		if state == StateRollback && info != s.hash {
			if err := gc.Rollback(info); err != nil {
				log.Warningf("Service %q, error rollback repo %q to %q: %s", s.Service, s.Upstream, info, err)
				s.SetState(StateDiff, fmt.Sprintf("error rolling back %q to %q: %s", s.Upstream, info, err))
				continue
			}
			if rerr := s.reload(); rerr != nil {
				log.Warningf("Service %q, error running systemctl daemon-reload: %s", s.Service, rerr)
				s.SetState(StateBroken, fmt.Sprintf("error running systemctl daemon-reload %q: %s", s.Upstream, rerr))
				continue
			} else if err := s.systemctl(); err != nil {
				log.Warningf("Service %q, error running systemctl: %s", s.Service, err)
				s.SetState(StateBroken, fmt.Sprintf("error running systemctl %q: %s", s.Upstream, err))
				continue
			}
			log.Warningf("Service %q, successfully rollback repo %q to %s", s.Service, s.Upstream, info)
			s.SetState(StateFreeze, "ROLLBACK: "+info)
			continue
		}

		if state, _ := s.State(); state == StateFreeze || state == StateRollback {
			log.Warningf("Service %q is in %s, not pulling", s.Service, state)
			continue
		}

		changed, err := gc.Pull()
		if err != nil {
			log.Warningf("Service %q, error pulling repo %q: %s", s.Service, s.Upstream, err)
			s.SetState(StateDiff, fmt.Sprintf("error pulling %q: %s", s.Upstream, err))
			continue
		}

		if !changed {
			continue
		}

		s.SetHash(gc.Hash())
		state, info = s.State()
		s.SetState(state, info)

		log.Infof("Service %q, diff in repo %q, pinging it", s.Service, s.Upstream)
		if rerr := s.reload(); rerr != nil {
			log.Warningf("Service %q, error running systemctl daemon-reload: %s", s.Service, rerr)
			s.SetState(StateBroken, fmt.Sprintf("error running systemctl daemon-reload %q: %s", s.Upstream, rerr))
			continue
		} else if err := s.systemctl(); err != nil {
			log.Warningf("Service %q, error running systemctl: %s", s.Service, err)
			s.SetState(StateBroken, fmt.Sprintf("error running systemctl %q: %s", s.Upstream, err))
			continue
		}
	}
}

func (s *Service) reload() error {
	ctx := context.TODO()
	cmd := exec.CommandContext(ctx, "systemctl", "daemon-reload", s.Service)
	log.Infof("running %v", cmd.Args)
	return cmd.Run()
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

func (s *Service) enable() error {
	ctx := context.TODO()
	cmd := exec.CommandContext(ctx, "systemctl", "enable", s.Service)
	log.Infof("running %v", cmd.Args)
	return cmd.Run()
}

// Boot returns the start time of the service. If that isn't available because there isn't a Service in s, then we
// return the kernel's boot time (i.e. when the system we started).
func (s *Service) SetBoot() {
	ctx := context.TODO()
	cmd := &exec.Cmd{}
	if s.Service != "" {
		cmd = exec.CommandContext(ctx, "systemctl", "show", "--property=ExecMainStartTimestamp", s.Service)
	} else {
		cmd = exec.CommandContext(ctx, "systemctl", "show", "--property=KernelTimestamp")
	}
	log.Infof("running %v", cmd.Args)
	out, err := cmd.Output()
	if err != nil {
		return
	}

	out = bytes.TrimSpace(out)
	// Testing show this is the string returned: Mon 2022-11-21 09:39:59 CET, so parse that into a time.Time
	t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", string(out))
	if err == nil { // on succes
		s.mu.Lock()
		defer s.mu.Unlock()
		s.stateStamp = t.UTC()
	}
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

		logtype := "Directory"
		if d.File {
			logtype = "File"
		}

		if !exists(d.Local) {
			switch d.File {
			case false:
				if err := os.MkdirAll(d.Local, 0775); err != nil {
					log.Errorf("Directory %q can not be created", d.Local)
					return 0, fmt.Errorf("failed to create directory %q: %s", d.Local, err)
				}
			case true:
				if err := os.MkdirAll(path.Dir(d.Local), 0775); err != nil {
					log.Errorf("Directory %q can not be created", path.Dir(d.Local))
					return 0, fmt.Errorf("failed to create directory %q: %s", path.Dir(d.Local), err)
				}
				file, err := os.Create(d.Local)
				if err != nil {
					log.Errorf("File %q, can not be created", d.Local)
					return 0, fmt.Errorf("failed to create file %q: %s", d.Local, err)
				}
				file.Close()

			}
			if os.Geteuid() == 0 { // set d.Local to the correct owner, if we are root
				uid, gid := osutil.User(s.User)
				if err := os.Chown(d.Local, int(uid), int(gid)); err != nil {
					log.Errorf("%s %q can not be chown-ed to %q: %s", logtype, d.Local, s.User, err)
					return 0, fmt.Errorf("failed to chown %s %q to %q: %s", strings.ToLower(logtype), d.Local, s.User, err)
				}
			}
		}

		if ok, err := mountinfo.Mounted(d.Local); err == nil && ok {
			log.Infof("%s %q is already mounted", logtype, d.Local)
			continue
		}

		ctx := context.TODO()
		cmd := exec.CommandContext(ctx, "mount", "--bind", gitdir, d.Local) // mount needs to be r/w for pkg upgrades
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

// jitter will add a random amount of jitter [0, d/2] to d.
func jitter(d time.Duration) time.Duration {
	rand.Seed(time.Now().UnixNano())
	max := d / 2
	return d + time.Duration(rand.Int63n(int64(max)))
}
