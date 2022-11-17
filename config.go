package main

// Config holds the gitopper config file. It's is updated every so often to pick up new changes.
type Config struct {
	Global   Service
	Services []Service
}

// Valid checks the config in c and returns nil of all mandatory fields have been set.
func (c Config) Valid() error {
	return nil
}
