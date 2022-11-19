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

	"go.science.ru.nl/log"
)

var flagHosts sliceFlag

var (
	flagConfig = flag.String("c", "", "config file to read")
	flagAddr   = flag.String("a", ":8000", "address to listen on")
	flagDebug  = flag.Bool("d", false, "enable debug logging")
)

func main() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	flag.Var(&flagHosts, "h", "hosts to impersonate, can be given multiple times, $HOSTNAME is included by default")
	(&flagHosts).Set(os.Getenv("HOSTNAME"))
	duration := 30 * time.Second
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
	for _, s := range c.Services {
		if !s.forMe(flagHosts) {
			continue
		}

		s1 := s.merge(c.Global, duration)
		log.Infof("Machine %q %q", s1.Machine, s1.Upstream)
		gc := s1.newGitCmd()

		// Initial checkout - if needed.
		err := gc.Checkout()
		if err != nil {
			log.Warningf("Machine %q, error pulling repo %q: %s", s.Machine, s.Upstream, err)
			s1.SetState(StateBroken, fmt.Sprintf("error pulling %q: %s", s.Upstream, err))
			continue
		}

		log.Infof("Machine %q, repository in %q with %q", s1.Machine, gc.Repo(), gc.Hash())

		// all succesfully done, do the bind mounts and start our puller
		if err := s1.bindmount(); err != nil {
			log.Warningf("Machine %q, error setting up bind mounts for %q: %s", s.Machine, s.Upstream, err)
			s1.SetState(StateBroken, fmt.Sprintf("error setting up bind mounts repo %q: %s", s.Upstream, err))
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s1.trackUpstream(ctx)
		}()
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
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
