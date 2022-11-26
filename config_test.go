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
	if _, err := parseConfig([]byte(conf)); err != nil {
		t.Fatalf("expected to parse config, but got: %s", err)
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
