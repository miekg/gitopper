package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/miekg/gitopper/log"
	toml "github.com/pelletier/go-toml/v2"
)

func main() {
	doc, err := os.ReadFile("config")
	if err != nil {
		log.Fatal(err)
	}

	var cfg Config
	if err := toml.Unmarshal([]byte(doc), &cfg); err != nil {
		log.Fatal(err)
	}
	// config check - abstract so we can do it seperately

	for _, s := range cfg.Services {
		s1 := s.merge(cfg.Global)

		log.Infof("Machine %q %q", s1.Machine, s1.Upstream)
		gc := s1.newGitCmd()

		// Initial checkout - if needed.
		err := gc.Checkout()
		if err != nil {
			log.Warningf("Machine %q, error checking out: %s", s1.Machine, err)
			// continue??
		}
		hash, _ := gc.Hash()
		log.Infof("Machine %q, repository in %q with %q", s1.Machine, gc.Repo(), hash)

		// all succesfully done, do the bind mounts and start our puller
		s1.bindmount()
		go s1.trackUpstream(nil)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}
