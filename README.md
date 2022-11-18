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

And things should work then. I.e. in /etc/grafana you should see the content of the
*miekg/blah-origin* repository.

The checked out git repo in /tmp/grafana1/grafana-server should _only_ contain the grafana directory
thanks to the sparse checkout. Changes made to the `crap` subdir in that repo do not trigger a
grafana restart (not even sure grafana actually needs a restart).

Then with cmd/gitopperctl/gitopperctl you can query the server a bit:

~~~
% ./gitopperctl list services @localhost
#  SERVICE         HASH     STATE  INFO  CHANGED
0  grafana-server  606eb57  OK           Fri, 18 Nov 2022 09:14:52 UTC
~~~

## Services

A service can be in 4 states: OK, FREEZE, ROLLBACK (which is a FREEZE to a previous commit) and
BROKEN.

These state are not carried over when gitopper crashes/stops (maybe we want this to be persistent,
would be nice to have this state in the git repo somehow?).

* `OK`: everything is running and we're tracking upstream.
* `FREEZE`: everything is running, but we're not tracking upstream.
* `ROLLBACK`: everything is running, but we're not tracking upstream *and* we're pinned to an older
  commit.
* `BROKEN`: something with the service is broken, we're still tracking upstream.

State transitions:


## Config File

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

* freeze a service to the current git commit
* unfreeze a service, i.e. to let it pull again

*do we need a freeze all for everything on the system?*

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

* TESTS
* Bootstrapping
* Reload config on the fly and re-initialize
* create target directory for bind mounts.... we should do this (correct user??)
* rollback
* set BROKEN
* check metrics
* query all machines, run a script
