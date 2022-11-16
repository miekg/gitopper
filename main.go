package main

import (
	"os"
	"path"

	"github.com/miekg/gitopper/gitcmd"
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
	for _, m := range cfg.Machines {
		m1 := merge(cfg.Global, m)
		log.Infof("Machine %q", m1.Name)

		dirs := []string{}
		for _, d := range m1.Dirs {
			dirs = append(dirs, d.Link)
		}

		repo := path.Join(m1.Mount, m1.Service)
		gc := gitcmd.New(m1.URL, repo, dirs)

		// Initial checkout
		err := gc.Checkout()
		if err != nil {
			log.Warningf("Machine %q, error checking out: %s", m1.Name, err)
		}
		hash, _ := gc.Hash()
		log.Infof("Machine %q, repository in %q with %q", m1.Name, repo, hash)
	}

	// Keep up to date

	// Listen to commands, cmdline rules.
}
