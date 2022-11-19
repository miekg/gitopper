package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/miekg/gitopper/osutil"
	"go.science.ru.nl/log"
)

var (
	flagHosts   sliceFlag
	flagConfig  = flag.String("c", "", "config file to read")
	flagAddr    = flag.String("a", ":8000", "address to listen on")
	flagDebug   = flag.Bool("d", false, "enable debug logging")
	flagRestart = flag.Bool("r", false, "receive SIGHUP when config changes (systemd should then restart us)")
)

func main() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	flag.Var(&flagHosts, "h", "hosts to impersonate, can be given multiple times, $HOSTNAME is included by default")
	duration := 30 * time.Second
	flagHosts.Set(osutil.Hostname())
	flag.Parse()

	if *flagDebug {
		log.D.Set()
	}

	if *flagConfig == "" {
		log.Fatalf("-c flag is mandatory")
	}

	doc, err := os.ReadFile(*flagConfig)
	if err != nil {
		log.Fatal(err)
	}
	c, err := parseConfig(doc)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Valid(); err != nil {
		log.Fatalf("The configuration is not valid: %s", err)
	}

	router := newRouter(c)
	go func() {
		// TODO: Interrupt HTTP serving through context cancellation.
		if err := http.ListenAndServe(*flagAddr, router); err != nil {
			log.Fatal(err)
		}
	}()
	log.Infof("Launched server on port %s", *flagAddr)

	var wg sync.WaitGroup
	for _, serv := range c.Services {
		if !serv.forMe(flagHosts) {
			continue
		}

		s := serv.merge(c.Global, duration)
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
	if *flagRestart {
		wg.Add(1)
		go func() {
			defer wg.Done()
			trackConfig(ctx, *flagConfig, done)
		}()
	}
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

func init() {
	h, err := os.Hostname()
	if err != nil {
		h = os.Getenv("HOSTNAME")
	}
	flagHosts.Set(h)
}
