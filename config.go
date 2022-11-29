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

	"github.com/gliderlabs/ssh"
	toml "github.com/pelletier/go-toml/v2"
	"go.science.ru.nl/log"
)

// Config holds the gitopper config file. It's is updated every so often to pick up new changes.
type Config struct {
	Global   `toml:"global"`
	Services []*Service
}

type Global struct {
	*Service
	Keys []*Key
}

type Key struct {
	Path          string
	RO            bool `toml:"ro"` // treat key as ro, and disallow "write" commands
	ssh.PublicKey `toml:"-"`
}

func parseConfig(doc []byte) (c Config, err error) {
	t := toml.NewDecoder(bytes.NewReader(doc))
	t.DisallowUnknownFields()
	err = t.Decode(&c)
	return c, err
}

// Valid checks the config in c and returns nil of all mandatory fields have been set.
func (c Config) Valid() error {
	if len(c.Global.Keys) == 0 {
		return fmt.Errorf("at least one public key should be specified")
	}

	for i, serv := range c.Services {
		s := serv.merge(c.Global)
		if s.Machine == "" {
			return fmt.Errorf("machine #%d, has empty machine name", i)
		}
		if s.Upstream == "" {
			return fmt.Errorf("machine #%d %q, has empty upstream", i, s.Machine)
		}
		if s.Mount == "" {
			return fmt.Errorf("machine #%d %q, has empty mount", i, s.Machine)
		}
		if s.Service == "" {
			return fmt.Errorf("machine #%d %q, has empty service", i, s.Service)
		}
	}
	return nil
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
