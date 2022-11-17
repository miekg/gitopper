package main

import "fmt"

// Config holds the gitopper config file. It's is updated every so often to pick up new changes.
type Config struct {
	Global   Service
	Services []Service
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
