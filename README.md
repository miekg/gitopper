# Gitopper

Watch a remote git repo, pull changes and HUP the service process. For a design doc see:
<https://miek.nl/2022/november/15/provisioning-services/>.

A sparse (but with full history) git checkout will be done, so each service will only see the files
it will actually need. Several bind mounts are then setup to give the service access to the file in
Git. If the target directories don't exist, they will be created, with the current user - if
specified.

## Features Implemented

From the doc:

> * metrics, so rollouts/updates can be tracked;
> * diff detection, so we know state doesn’t reconcile;
> * out of band rollbacks;
> * no client side processing;
> * canarying.

- *Metrics*: are included see below, they export a Git hash, so a rollout can be tracked.
- *Diff detection*: possible using the metrics or gitopperctl.
- *Out of band rollbacks*: use gitopperctl to bypass the normal Git workflow.
- *No client side processing*: files are used as the are in the Git repo.
- *Canarying*: give a service a different branch to checkout.

## Quickstart

- Install grafana OSS version from the their website (just using this as a test case, nothing
  special here)
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
#  SERVICE         HASH                                     STATE  INFO  CHANGED
0  grafana-server  606eb576c1b91248e4c1c4cd0d720f27ac0deb70 OK           Fri, 18 Nov 2022 09:14:52 UTC
~~~

## Services

A service can be in 4 states: OK, FREEZE, ROLLBACK (which is a FREEZE to a previous commit) and
BROKEN.

These state are not carried over when gitopper crashes/stops (maybe we want this to be persistent,
would be nice to have this state in the git repo somehow?).

* `OK`: everything is running and we're tracking upstream.
* `FREEZE`: everything is running, but we're not tracking upstream.
* `ROLLBACK`: everything is running, but we're not tracking upstream *and* we're pinned to an older
  commit. This state is quickly followed by FREEZE if we were successful rolling back, otherwise
  BROKEN.
* `BROKEN`: something with the service is broken, we're still tracking upstream.

ROLLBACK is a transient state and quickly moves to FREEZE, unless something goes wrong then it
becomes BROKEN.

## Config File

~~~ toml
[global]
upstream = "https://github.com/miekg/blah-origin"  # repository where to download from
mount = "/tmp"                                     # directory where to download to, mount+service is used as path

[[services]]
machine = "grafana.atoom.net" # hostname of the machine, so a host knows when to pick this up.
branch = "main"               # what branch to checkout
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
* rollback a service to a specific commit

routes.go defines where the metrics live.

## Metrics

The following metrics are exported:

* gitopper_service_info{"service", "hash", "state"}, where 'hash' is unbounded, but we really care
  about that value.
* gitopper_machine_git_error_total - total number of errors when running git.
* gitopper_machine_git_ops_total - total number of git runs.

Metrics are available under the /metrics endpoint.

## Client

A client is included in cmd/gitopperctl. It has it's own README.md.

## Authentication

TODO...? Some plugins based solution?

## TODO

* Tests
  - config
* Bootstrapping
* Reload config on the fly and re-initialize?? Or just restart the binary? Might actually be cleaner
  and easier.
* Start webservice before doing checkouts?
* authentication for destructive action
* TLS
