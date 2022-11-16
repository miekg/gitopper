package main

import (
	"fmt"
	"log"
	"os"

	"github.com/miekg/gitopper/gitcmd"
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
	for i, m := range cfg.Machines {
		m1 := merge(cfg.Global, m)
		fmt.Printf("%d %+v\n", i, m1)

		dirs := []string{}
		for _, d := range m1.Dirs {
			dirs = append(dirs, d.Link)
		}

		gc := gitcmd.New(m1.URL, m1.Mount, dirs)

		// Initial checkout
		err := gc.Checkout()
		if err != nil {
			log.Printf("Error checking out: %s", err)
		}
	}

	// Keep up to date

	// Listen to commands, cmdline rules.
}
