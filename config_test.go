package main

import (
	"testing"
)

func TestValidConfig(t *testing.T) {
	const conf = `
[global]
upstream = "https://github.com/miekg/gitopper-config"
mount = "/tmp"

[[services]]
machine = "localhost"
branch = "main"
service = "prometheus"
user = "grafana"
package = "grafana"
action = "reload"
dirs = [
    { local = "/etc/prometheus", link = "prometheus/etc" },
]
`
	c, err := parseConfig([]byte(conf))
	if err != nil {
		t.Fatalf("expected to parse config, but got: %s", err)
	}
	serv := c.Services[0]
	serv = serv.merge(c.Global)
	if serv.Mount == "" {
		t.Errorf("expected service to have Mount, got none")
	}
	if serv.Upstream == "" {
		t.Errorf("expected service to have Upstream, got none")
	}
	if serv.Branch != "canary" {
		t.Errorf("expected Branch to be %s, got %s", "canary", serv.Branch)
	}
}

func TestInvalidConfig(t *testing.T) {
	const conf = `
[global]
upstream = "https://github.com/miekg/gitopper-config"
mount = "/tmp"

[[services]]
machine = "localhost"
brokenbranch = "main"
service = "prometheus"
`
	if _, err := parseConfig([]byte(conf)); err == nil {
		t.Fatalf("expected to fail to parse config, but got nil error")
	}
}
