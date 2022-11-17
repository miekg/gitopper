package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	toml "github.com/pelletier/go-toml/v2"
	"go.science.ru.nl/log"
)

var (
	flagConfig   = flag.String("c", "", "config file to read")
	flagDuration = flag.String("d", "30s", "default duration to sleep before pulling")
)

func main() {
	flag.Parse()

	if *flagConfig == "" {
		log.Fatalf("-c flag is mandatory")
	}
	duration, err := time.ParseDuration(*flagDuration)
	if err != nil {
		log.Fatalf("invalid -d value: %s", err)
	}
	if duration < 5*time.Second {
		log.Fatal("invalid -d value (<5s)")
	}

	doc, err := os.ReadFile(*flagConfig)
	if err != nil {
		log.Fatal(err)
	}

	var c Config
	if err := toml.Unmarshal([]byte(doc), &c); err != nil {
		log.Fatal(err)
	}

	if err := c.Valid(); err != nil {
		log.Fatalf("The configuration is not valid: %s", err)
	}
	// config check - abstract so we can do it seperately
	// start webservies

	for _, s := range c.Services {
		s1 := s.merge(c.Global, duration)

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
