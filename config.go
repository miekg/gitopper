package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	toml "github.com/pelletier/go-toml/v2"
	"go.science.ru.nl/log"
)

// Config holds the gitopper config file. It's is updated every so often to pick up new changes.
type Config struct {
	Global   *Service
	Services []*Service
}

func parseConfig(doc []byte) (c Config, err error) {
	t := toml.NewDecoder(bytes.NewReader(doc))
	t.DisallowUnknownFields()
	err = t.Decode(&c)
	return c, err
}

// trackConfig will sha1 sum the contents of file and if it differs from previous runs, will SIGHUP ourselves so we
// exist with status code 2, which in turn will systemd restart us again.
func trackConfig(ctx context.Context, file string, done chan os.Signal) {
	hash := ""
	for {
		select {
		case <-time.After(30 * time.Second):
		case <-ctx.Done():
			return
		}
		doc, err := ioutil.ReadFile(file)
		if err != nil {
			log.Warningf("Failed to read config %q: %s", file, err)
			continue
		}
		sha := sha1.New()
		sha.Write(doc)
		hash1 := string(sha.Sum(nil))
		if hash == "" {
			hash = hash1
			continue
		}
		if hash1 != hash {
			log.Info("Config change detected, sending SIGHUP")
			// haste our exit (can this block?)
			done <- syscall.SIGHUP
			return
		}
	}
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
