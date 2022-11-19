package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/miekg/gitopper/osutil"
	"go.science.ru.nl/log"
)

var flagHosts sliceFlag

var (
	flagConfig = flag.String("c", "", "config file to read")
	flagAddr   = flag.String("a", ":8000", "address to listen on")
	flagDebug  = flag.Bool("d", false, "enable debug logging")
	// bootstrap flags
	flagUpstream = flag.String("U", "", "when bootstrapping use this git repo")
	flagDir      = flag.String("D", "gitopper", "directory to sparse checkout")
	flagBranch   = flag.String("B", "main", "when bootstrapping check out in this branch")
	flagMount    = flag.String("M", "", "when bootstrapping use this dir as mount, if given -c is relative to this dir")
)

func main() {
	flag.Var(&flagHosts, "h", "hosts to impersonate, can be given multiple times, local hostname is included by default")
	(&flagHosts).Set(osutil.Hostname())
	duration := 30 * time.Second
	flag.Parse()

	if *flagDebug {
		log.D.Set()
	}
	if *flagConfig == "" {
		log.Fatalf("-c flag is mandatory")
	}

	// Adding ourselves in a local git repo fails with 'dubious ownership' because gitopper needs root, and the repo
	// is checked out by Joe User. So we only do this when the bootstrap flags are given. In that case we add
	// ourselves to the managed services managed.
	self := selfService(*flagUpstream, *flagBranch, *flagMount, *flagDir)
	if self != nil {
		gc := self.newGitCmd()
		// Initial checkout - if needed.
		err := gc.Checkout()
		if err != nil {
			log.Warningf("Machine %q, error pulling repo %q: %s", self.Machine, self.Upstream, err)
			self.SetState(StateBroken, fmt.Sprintf("error pulling %q: %s", self.Upstream, err))
			// no we have a problem.... I think this is a fatal error
		}
		*flagConfig = path.Join(path.Join(self.Mount, self.Service), *flagConfig)
		log.Infof("Setting config to %s", *flagConfig)
	}

	doc, err := os.ReadFile(*flagConfig)
	if err != nil {
		log.Fatalf("Failed to read config %q: %s", *flagConfig, err)
	}
	c, err := parseConfig(doc)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Valid(); err != nil {
		log.Fatalf("The configuration is not valid: %s", err)
	}

	router := newRouter(&c)
	go func() {
		// TODO: Interrupt HTTP serving through context cancellation.
		if err := http.ListenAndServe(*flagAddr, router); err != nil {
			log.Fatal(err)
		}
	}()
	log.Infof("Launched server on port %s", *flagAddr)

	if self != nil {
		log.Infof("Adding ourselves to managed services, for machine %s", self.Machine)
		// this need locking
		c.Services = append(c.Services, self)
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	var wg sync.WaitGroup
	for _, s := range c.Services {
		if !s.forMe(flagHosts) {
			continue
		}

		s := s.merge(c.Global, duration)
		log.Infof("Machine %q %q", s.Machine, s.Upstream)
		gc := s.newGitCmd()

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

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	wg.Add(1)
	go func() {
		defer wg.Done()
		trackConfig(ctx, *flagConfig, done)
	}()

	go func() {
		select {
		case s := <-done:
			cancel()
			// on HUP exit with exit status 2, so systemd can restart us (Restart=OnFailure)
			if s == syscall.SIGHUP {
				defer os.Exit(2)
			}
		case <-ctx.Done():
		}
	}()
	wg.Wait()
}
