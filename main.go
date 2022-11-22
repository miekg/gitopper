package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/gliderlabs/ssh"
	"github.com/miekg/gitopper/ospkg"
	"github.com/miekg/gitopper/osutil"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.science.ru.nl/log"
)

type ExecContext struct {
	Hosts        []string
	ConfigSource string
	SAddr        string
	MAddr        string
	Debug        bool
	Restart      bool
	Upstream     string
	Dir          string
	Branch       string
	Mount        string
	Pull         bool
}

func (exec *ExecContext) RegisterFlags(fs *flag.FlagSet) {
	if fs == nil {
		fs = flag.CommandLine
	}
	fs.Var(&sliceFlag{&exec.Hosts}, "h", "hosts to impersonate, can be given multiple times, $HOSTNAME is included by default")
	fs.StringVar(&exec.ConfigSource, "c", "", "config file to read")
	fs.StringVar(&exec.SAddr, "s", ":2222", "ssh address to listen on")
	fs.StringVar(&exec.MAddr, "m", ":9222", "http metrics address to listen on")
	fs.BoolVar(&exec.Debug, "d", false, "enable debug logging")
	fs.BoolVar(&exec.Restart, "r", false, "send SIGHUP when config changes")

	// bootstrap flags
	fs.StringVar(&exec.Upstream, "U", "", "[bootstrapping] use this git repo")
	fs.StringVar(&exec.Dir, "D", "gitopper", "[bootstrapping] directory to sparse checkout")
	fs.StringVar(&exec.Branch, "B", "main", "[bootstrapping] check out in this branch")
	fs.StringVar(&exec.Mount, "M", "", "[bootstrapping] check out into this directory, -c is relative to this dir")
	fs.BoolVar(&exec.Pull, "P", false, "[boostrapping] pull (update) the git repo to the newest version before starting")
}

var (
	ErrNotRoot  = errors.New("not root")
	ErrNoConfig = errors.New("-c flag is mandatory")
	ErrHUP      = errors.New("hangup requested")
)

type RepoPullError struct {
	Machine    string
	Upstream   string
	Underlying error
}

func (err *RepoPullError) Error() string {
	return fmt.Sprintf("Machine %q, error pulling repo %q: %s", err.Machine, err.Upstream, err.Underlying)
}

func (err *RepoPullError) Unwrap() error { return err.Underlying }

func run(exec *ExecContext) error {
	if os.Geteuid() != 0 {
		return ErrNotRoot
	}

	if exec.Debug {
		log.D.Set()
	}

	if exec.ConfigSource == "" {
		return ErrNoConfig
	}

	// bootstrapping
	self := selfService(exec.Upstream, exec.Branch, exec.Mount, exec.Dir)
	if self != nil {
		log.Infof("Bootstapping from repo %q and adding service %q for %q", exec.Upstream, self.Service, self.Machine)
		gc := self.newGitCmd()
		err := gc.Checkout()
		if err != nil {
			return &RepoPullError{self.Machine, self.Upstream, err}
		}
		if exec.Pull {
			if _, err := gc.Pull(); err != nil {
				return &RepoPullError{self.Machine, self.Upstream, err}
			}
		}
		exec.ConfigSource = path.Join(path.Join(path.Join(self.Mount, self.Service), exec.Dir), exec.ConfigSource)
		log.Infof("Setting config to %s", exec.ConfigSource)
	}

	doc, err := os.ReadFile(exec.ConfigSource)
	if err != nil {
		return fmt.Errorf("reading config: %v", err)
	}
	c, err := parseConfig(doc)
	if err != nil {
		return fmt.Errorf("parsing config: %v", err)
	}

	if err := c.Valid(); err != nil {
		return fmt.Errorf("validating config: %v", err)
	}

	if self != nil {
		c.Services = append(c.Services, self)
	}

	allowed := make([]ssh.PublicKey, len(c.Keys))
	for i, k := range c.Keys {
		if !path.IsAbs(k.Path) && self != nil { // bootstrapping
			newpath := path.Join(path.Join(path.Join(self.Mount, self.Service), exec.Dir), k.Path)
			k.Path = newpath
		}

		log.Infof("Reading public key %q", k.Path)
		data, err := ioutil.ReadFile(k.Path)
		if err != nil {
			log.Fatal(err)
		}
		a, _, _, _, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			log.Fatal(err)
		}
		allowed[i] = a
	}

	newRouter(c, exec.Hosts)
	go func() {
		// TODO: Interrupt SSH serving through context cancellation.
		ssh.ListenAndServe(exec.SAddr, nil,
			ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
				for _, a := range allowed {
					if ssh.KeysEqual(a, key) {
						return true
					}
				}
				return false
			}),
		)
	}()
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		// TODO: Interrupt HTTP serving through context cancellation.
		if err := http.ListenAndServe(exec.MAddr, nil); err != nil {
			log.Fatal(err)
		}
	}()
	log.Infof("Launched servers on port %s and %s from machines: %v", exec.SAddr, exec.MAddr, exec.Hosts)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	pkg := ospkg.New()

	var wg sync.WaitGroup
	servCnt := 0
	for _, serv := range c.Services {
		if !serv.forMe(exec.Hosts) {
			continue
		}

		servCnt++
		s := serv.merge(c.Global)
		log.Infof("Machine %q %q", s.Machine, s.Upstream)
		gc := s.newGitCmd()

		if s.Package != "" {
			if err := pkg.Install(s.Package); err != nil {
				log.Warningf("Machine %q, error installing package %q: %s", s.Machine, s.Package, err)
				continue // skip this, or continue, if continue and with the bind mounts the future pkg install might also break...
				// or fatal error??
			}
		}

		// Initial checkout - if needed.
		err := gc.Checkout()
		if err != nil {
			log.Warningf("Machine %q, error pulling repo %q: %s", s.Machine, s.Upstream, err)
			s.SetState(StateBroken, fmt.Sprintf("error pulling %q: %s", s.Upstream, err))
			continue
		}

		log.Infof("Machine %q, repository in %q with %q", s.Machine, gc.Repo(), gc.Hash())

		// all succesfully done, do the bind mounts and start our puller
		mounts, err := s.bindmount()
		if err != nil {
			log.Warningf("Machine %q, error setting up bind mounts for %q: %s", s.Machine, s.Upstream, err)
			s.SetState(StateBroken, fmt.Sprintf("error setting up bind mounts repo %q: %s", s.Upstream, err))
			continue
		}
		// Restart any services as they see new files in their bindmounts. Do this here, because we can't be
		// sure there is an update to a newer commit that would also kick off a restart.
		if mounts > 0 {
			if err := s.systemctl(); err != nil {
				log.Warningf("Machine %q, error running systemctl: %s", s.Machine, err)
				s.SetState(StateBroken, fmt.Sprintf("error running systemctl %q: %s", s.Upstream, err))
				// no continue; maybe git pull will make this work later
			}
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.trackUpstream(ctx)
		}()
	}

	if servCnt == 0 {
		log.Warningf("No services found for machine: %v, exiting", exec.Hosts)
		return nil
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	if exec.Restart {
		wg.Add(1)
		go func() {
			defer wg.Done()
			trackConfig(ctx, exec.ConfigSource, done)
		}()
	}
	hup := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case s := <-done:
			cancel()
			if s == syscall.SIGHUP {
				close(hup)
			}
		case <-ctx.Done():
		}
	}()
	wg.Wait()
	select {
	case <-hup:
		return ErrHUP
	default:
	}
	return nil
}

func main() {
	exec := ExecContext{Hosts: []string{osutil.Hostname()}}
	exec.RegisterFlags(nil)

	flag.Parse()
	err := run(&exec)
	switch {
	case err == nil:
	case errors.Is(err, ErrHUP):
		// on HUP exit with exit status 2, so systemd can restart us (Restart=OnFailure)
		os.Exit(2)
	default:
		log.Fatal(err)
	}
}
