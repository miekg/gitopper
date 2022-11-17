# Gitopper

Watch a remote git repo, pull changes and HUP the service process. For a design doc see:
<https://miek.nl/2022/november/15/provisioning-services/>.

A sparse (but with full history) git checkout will be done, so each service will only see the files
it will actually need. Several bind mounts are then setup to give the service access to the file in
Git.

If the target directories for the bind mounts don't exist they will not (yet) be created. This is
now also a fatal error.

## Quickstart

- Install grafana OSS version from the their website (just using this as a test case, nothing
  special here)
- `mkdir /etc/grafana` and `mkdir -p /var/lib/grafana/dashboards` (see above)
- Compile the gitopper binary: `go build`
- Start as root: `sudo ./gitopper -c config`

And things should work then. I.e. in /etc/grafana you should see the contant of the
*miekg/blah-origin* repository.

The checked out git repo in /tmp/grafana1/grafana-server should _only_ contain the grafana directory
thanks to the sparse checkout. Changes made to the `crap` subdir in that repo do not trigger a
grafana restart (not even sure grafana actually needs this).

## Config file

~~~ toml
[global]
upstream = "https://github.com/miekg/blah-origin"  # repository where to download from
mount = "/tmp"                                     # directory where to download to, mount+service is used as path

[[services]]
machine = "grafana.atoom.net" # hostname of the machine, so a host knows when to pick this up.
service = "grafana-server"    # service identifier, if it's used by systemd it must be the systemd service name
package = "grafana"           # as used by package mgmt, may be empty (not implemented yet)
user = "grafana"              # do the checkout with this user
action = "reload"             # call systemctl <action> <service> when the git repo changes.
mount = "/tmp/grafana1"       # where to put the downloaded download (we don't care - might be removed)
dirs = [
    { local = "/etc/grafana", link = "grafana/etc" },
    { local = "/var/lib/grafana/dashboards", link = "grafana/dashboards" }
]
~~~

### REST

See proto/proto.go for the defined interface. Interaction is REST, thus JSON. You can

* list all defined machines
* list services run on this host
* list a specific service

## Client

A client is included in cmd/gitopperctl. It has it's own README.md.

## Notes and Crap

TODO in the ctl:

gitopperctl freeze @machine <service-name> [<duration>]
    /state/freeze/<name>?dur=5s

gitopperctl unfreeze @machine <service-name>
    /state/unfreeze/<name>

gitopperctl rollback @machine <service-name> <hash>
    /state/rollback/<name>?to=<hash>

gitoppercli redo @machine <service-name>
    /state/redo/<name>

## Authentication

...

## metrics

~~~ txt
gitopper_machine_info{machine="..."} 1.0
gitopper_service_hash{service="...", hash="..."} 1.0
gitopper_service_frozen{service="..."} 1.0
gitopper_service_ok{service="..."} 1.0
gitopper_service_failure_count{} 1.0
~~~

Some are implemented under the /metrics endpoint.

## TODO

* TESTST
* Bootstrapping
* Reload config on the fly and re-initialize then
* create target directory for bind mounts.... package should do this.
* add: -h options, multiple hostnames, for testing and other purposes.
