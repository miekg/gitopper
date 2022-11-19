package main

import (
	"testing"
)

func TestValidConfig(t *testing.T) {
	const conf = `
[global]
upstream = "https://github.com/miekg/blah-origin"
mount = "/tmp"

[[services]]
machine = "grafana.atoom.net"
branch = "main"
service = "grafana-server"
user = "grafana"
package = "grafana"
action = "restart"
mount = "/tmp/grafana1"
dirs = [
    { local = "/etc/grafana", link = "grafana/etc" },
    { local = "/var/lib/grafana/dashboards", link = "grafana/dashboards" }
]
`
	if _, err := parseConfig([]byte(conf)); err != nil {
		t.Fatalf("expected to parse config, but got: %s", err)
	}
}

func TestInvalidConfig(t *testing.T) {
	const conf = `
[global]
upstream = "https://github.com/miekg/blah-origin"
mount = "/tmp"

[[services]]
machine = "grafana.atoom.net"
brokenbranch = "main"
service = "grafana-server"
`
	if _, err := parseConfig([]byte(conf)); err == nil {
		t.Fatalf("expected to fail to parse config, but got nil error")
	}
}
