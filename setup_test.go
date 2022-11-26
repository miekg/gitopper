package main

import (
	"os"
	"testing"

	"go.science.ru.nl/log"
)

func TestInitialGitCheckout(t *testing.T) {
	log.Discard()
	temp, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		t.Fatalf("Failed to make temp dir: %q: %s", temp, err)
	}
	defer func() { os.RemoveAll(temp) }()
	s := Service{
		Upstream: "https://github.com/miekg/gitopper-config",
		Branch:   "main",
		Service:  "test-service",
		Mount:    temp,
		Dirs: []Dir{
			{Link: "grafana/etc"},
			{Link: "grafana/dashboards"},
		},
	}

	gc := s.newGitCmd()
	if err := gc.Checkout(); err != nil {
		t.Fatalf("Failed to checkout repo %q in %s: %s", s.Upstream, temp, err)
	}
}
