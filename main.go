package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/miekg/gitopper/ospkg"
	"github.com/miekg/gitopper/osutil"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"
	"go.science.ru.nl/log"
)

type ExecContext struct {
	// Configuration
	Hosts        []string
	ConfigSource string
	SAddr        string
	MAddr        string
	Debug        bool
	Restart      bool
	Root         bool
	Duration     time.Duration
	Upstream     string
	Dir          string
	Branch       string
	Mount        string
	Pull         bool

	// Runtime State
	HTTPMux *http.ServeMux
}

func (exec *ExecContext) RegisterFlags(fs *flag.FlagSet) {
	if fs == nil {
		fs = flag.CommandLine
	}
	fs.SortFlags = false
	fs.StringSliceVarP(&exec.Hosts, "hosts", "h", []string{osutil.Hostname()}, "hosts (comma separated) to impersonate, hostname is always added")
	fs.StringVarP(&exec.ConfigSource, "config", "c", "", "config file to read")
	fs.StringVarP(&exec.SAddr, "ssh", "s", ":2222", "ssh address to listen on")
	fs.StringVarP(&exec.MAddr, "metric", "m", ":9222", "http metrics address to listen on")
	fs.BoolVarP(&exec.Debug, "debug", "d", false, "enable debug logging")
	fs.BoolVarP(&exec.Restart, "restart", "r", false, "send SIGHUP when config changes")
	fs.BoolVarP(&exec.Root, "root", "o", true, "require root permission, setting to false can aid in debugging")
	fs.DurationVarP(&exec.Duration, "duration", "t", 5*time.Minute, "default duration between pulls")

	// bootstrap flags
	fs.StringVarP(&exec.Upstream, "upstream", "U", "", "[bootstrapping] use this git repo")
	fs.StringVarP(&exec.Dir, "directory", "D", "gitopper", "[bootstrapping] directory to sparse checkout")
	fs.StringVarP(&exec.Branch, "branch", "B", "main", "[bootstrapping] check out in this branch")
	fs.StringVarP(&exec.Mount, "mount", "M", "", "[bootstrapping] check out into this directory, -c is relative to this directory")
	fs.BoolVarP(&exec.Pull, "pull", "P", false, "[bootstrapping] pull (update) the git repo to the newest version before starting")
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

func (err *RepoPullError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Underlying
}

func serveMonitoring(exec *ExecContext, controllerWG, workerWG *sync.WaitGroup) error {
	exec.HTTPMux.Handle("/metrics", promhttp.Handler())
	ln, err := net.Listen("tcp", exec.MAddr)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:    exec.MAddr,
		Handler: exec.HTTPMux,
	}
	controllerWG.Add(1) // Ensure  HTTP server draining blocks application shutdown.
	go func() {
		defer controllerWG.Done()
		workerWG.Wait()              // Unblocks upon context cancellation and workers finishing.
		srv.Shutdown(context.TODO()) // TODO: Derive context tree more carefully from root.
	}()
	controllerWG.Add(1)
	go func() {
		defer controllerWG.Done()
		err := srv.Serve(ln)
		switch {
		case err == nil:
		case errors.Is(err, http.ErrServerClosed):
		default:
			log.Fatal(err)
		}
	}()
	return nil
}

func serveSSH(exec *ExecContext, controllerWG, workerWG *sync.WaitGroup, allowed []*Key, sshHandler ssh.Handler) error {
	l, err := net.Listen("tcp", exec.SAddr)
	if err != nil {
		return err
	}
	srv := &ssh.Server{Addr: exec.SAddr, Handler: sshHandler}
	srv.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		for _, a := range allowed {
			if ssh.KeysEqual(a.PublicKey, key) {
				log.Infof("Granting access for user %q with public key %q", ctx.User(), a.Path)
				return true
			}
		}
		log.Warningf("No valid keys found for user %q", ctx.User())
		return false
	}))
	controllerWG.Add(1) // Ensure SSH server draining blocks application shutdown.
	go func() {
		defer controllerWG.Done()
		workerWG.Wait()              // Unblocks upon context cancellation and workers finishing.
		srv.Shutdown(context.TODO()) // TODO: Derive context tree more carefully from root.
	}()
	controllerWG.Add(1)
	go func() {
		defer controllerWG.Done()
		err := srv.Serve(l)
		switch {
		case err == nil:
		case errors.Is(err, ssh.ErrServerClosed):
		default:
			log.Fatal(err)
		}
	}()
	return nil
}

func run(exec *ExecContext) error {
	if os.Geteuid() != 0 && exec.Root {
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
		log.Infof("Bootstrapping from repo %q and adding service %q for %q", exec.Upstream, self.Service, self.Machine)
		gc := self.newGitCmd()
		err := gc.Checkout()
		if err != nil {
			return &RepoPullError{self.Machine, self.Upstream, err}
		}
		if exec.Pull {
			if _, err := gc.Pull(); err != nil {
				// don't exit here, we have a repo, maybe it's good enough, we can always pull later
				log.Warningf("Bootstrapping service %q, error pulling repo %q: %s, continuing", self.Service, self.Upstream, err)
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

	for _, k := range c.Global.Keys {
		if !path.IsAbs(k.Path) && self != nil { // bootstrapping
			newpath := path.Join(path.Join(path.Join(self.Mount, self.Service), exec.Dir), k.Path)
			k.Path = newpath
		}

		log.Infof("Reading public key %q", k.Path)
		data, err := ioutil.ReadFile(k.Path)
		if err != nil {
			return err
		}
		a, _, _, _, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			return err
		}
		k.PublicKey = a
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	var workerWG, controllerWG sync.WaitGroup
	defer controllerWG.Wait()

	// start a fake worker thread, that in the case of no actual threads, will call done on the workerWG (and more
	// importantly will now have seen at least one Add(1)). This will make sure the serveMetrics and serveSSH return
	// correctly on receiving ^C.
	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		select {
		case <-ctx.Done():
			return
		}
	}()

	pkg := ospkg.New()
	servCnt := 0
	hostServices := map[string]struct{}{} // we can't have duplicate service name on a single machine.
	for _, serv := range c.Services {
		if !serv.forMe(exec.Hosts) {
			continue
		}
		if _, ok := hostServices[serv.Service]; ok {
			log.Fatalf("Service %q has a duplicate on these machines %v", serv.Service, exec.Hosts)
		}
		hostServices[serv.Service] = struct{}{}

		servCnt++
		s := serv.merge(c.Global)
		log.Infof("Service %q with upstream %q", s.Service, s.Upstream)
		gc := s.newGitCmd()

		if s.Package != "" {
			if err := pkg.Install(s.Package); err != nil {
				log.Warningf("Service %q, error installing package %q: %s", s.Service, s.Package, err)
				continue // skip this, or continue, if continue and with the bind mounts the future pkg install might also break...
				// or fatal error??
			}
		}

		// Initial checkout - if needed.
		err := gc.Checkout()
		if err != nil {
			log.Warningf("Service %q, error pulling repo %q: %s", s.Service, s.Upstream, err)
			s.SetState(StateBroken, fmt.Sprintf("error pulling %q: %s", s.Upstream, err))
			continue
		}

		log.Infof("Service %q, repository in %q with %q", s.Service, gc.Repo(), gc.Hash())

		// all succesfully done, do the bind mounts and start our puller
		mounts, err := s.bindmount()
		if err != nil {
			log.Warningf("Service %q, error setting up bind mounts for %q: %s", s.Service, s.Upstream, err)
			s.SetState(StateBroken, fmt.Sprintf("error setting up bind mounts repo %q: %s", s.Upstream, err))
			continue
		}
		// Restart any services as they see new files in their bindmounts. Do this here, because we can't be
		// sure there is an update to a newer commit that would also kick off a restart.
		if mounts > 0 {
			if err := s.systemctl(); err != nil {
				log.Warningf("Service %q, error running systemctl: %s", s.Service, err)
				s.SetState(StateBroken, fmt.Sprintf("error running systemctl %q: %s", s.Upstream, err))
				// no continue; maybe git pull will make this work later
			}
		}

		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			s.trackUpstream(ctx, exec.Duration)
		}()
	}

	if servCnt == 0 {
		log.Warningf("No services found for machine: %v, exiting", exec.Hosts)
		return nil
	}
	sshHandler := newRouter(c, exec.Hosts)
	if err := serveSSH(exec, &controllerWG, &workerWG, c.Global.Keys, sshHandler); err != nil {
		return err
	}
	if err := serveMonitoring(exec, &controllerWG, &workerWG); err != nil {
		return err
	}
	log.Infof("Launched servers on port %s (ssh) and %s (metrics) for machines: %v, %d public keys loaded", exec.SAddr, exec.MAddr, exec.Hosts, len(c.Global.Keys))

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	if exec.Restart {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			trackConfig(ctx, exec.ConfigSource, done)
		}()
	}
	hup := make(chan struct{})
	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		select {
		case s := <-done:
			cancel()
			if s == syscall.SIGHUP {
				close(hup)
			}
		case <-ctx.Done():
		}
	}()
	workerWG.Wait()
	select {
	case <-hup:
		return ErrHUP
	default:
	}
	return nil
}

func main() {
	exec := ExecContext{HTTPMux: http.NewServeMux()}
	exec.RegisterFlags(nil)
	flag.Parse()
	flag.VisitAll(func(f *flag.Flag) {
		// add hostname if not already there
		if f.Name != "hosts" {
			return
		}
		ok := false
		hostname := osutil.Hostname()
		for _, v := range f.Value.(flag.SliceValue).GetSlice() {
			if v == hostname {
				ok = true
				break
			}
		}
		if !ok {
			f.Value.Set(osutil.Hostname())
		}
	})
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
