package main

import (
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

// Config holds the gitopper config file. It's is updated every so often to pick up new changes.
type Config struct {
	Global   *Service
	Services []*Service
}

func parseConfig(file string) (Config, error) {
	var c Config

	doc, err := os.ReadFile(file)
	if err != nil {
		return c, err
	}

	err = toml.Unmarshal([]byte(doc), &c)
	return c, err
}

// Valid checks the config in c and returns nil of all mandatory fields have been set.
func (c Config) Valid() error {
	for i, s := range c.Services {
		s1 := s.merge(c.Global, 0) // don't care about duration here
		if s1.Machine == "" {
			return fmt.Errorf("machine #%d, has empty machine name", i)
		}
		if s1.Upstream == "" {
			return fmt.Errorf("machine #%d %q, has empty upstream", i, s1.Machine)
		}
		if s1.Mount == "" {
			return fmt.Errorf("machine #%d %q, has empty mount", i, s1.Machine)
		}
		if s1.Service == "" {
			return fmt.Errorf("machine #%d %q, has empty service", i, s1.Service)
		}
	}
	return nil
}
