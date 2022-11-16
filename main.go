package main

import (
	"os"

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
		s1 := merge(cfg.Global, s)
		log.Infof("Machine %q", s1.Machine)

		gc := NewGitCmd(s1)

		// Initial checkout
		err := gc.Checkout()
		if err != nil {
			log.Warningf("Machine %q, error checking out: %s", s1.Machine, err)
		}
		hash, _ := gc.Hash()
		log.Infof("Machine %q, repository in %q with %q", s1.Machine, repo, hash)
	}
}
