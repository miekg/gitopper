# Gitopper

Watch a remote git repo, pull changes and HUP the service process. For a design doc see:
<https://miek.nl/2022/november/15/provisioning-services/>.

A sparse (but with full history) git checkout will be done, so each service will only see the files
it will actually need.

## Bootstrapping


## Config file

~~~ toml
[global]
upstream = "https://github.com/miekg/blah-origin"  # repository where to download from
mount = "/tmp"                                     # directory where to download to, mount+service is used as path

[[services]]
machine = "grafana.atoom.net" # hostname of the machine, so it know what to do there.
service = "grafana-server" # service identifier, if it's used by systemd it must be the systemd service name
package = "grafana"  # as used by package mgmt, may be empty
action = "reload"    # what to do when files are changed, maybe empty.
mount = "/tmp/grafana1" # where to download the repo - we don't care
dirs = [
    { local = "/etc/grafana", link = "grafana/etc", single = false },
    { local = "/var/lib/grafana/dashboards", link = "grafana/dashboards", single = false }
]
~~~

## Interfaces

metrics, rest-like interface, return json, make client show it nicely.

## "Admin" Interface

- list all machines - from the config file?
- list all services from a machine - should be from the config as well...
- then for a service:
    * freeze ?duration=50
    * rollback ?to=<hash> (can error); rollback also freezes, rollback isn't a permanent state.
    * unfreeze
- get status of services, check also upstream repo

curl essentially

basic auth, then system auth... account exists... and in special group??

## Client

Included is a little client, called `gitopper-cli` that eases remote interaction with the server's
REST interface.

-ctl is basically a curl client, but make a client, simple as hell

Generic options. -u user -p <passwd>

These get json back:

gitopper-ctl list machines
gitopper-ctl list services @machine
gitopper-ctl status <service-name> @machine  git, pull, last update??

These are just posts, without json reply, just HTTP status code... with json payload??

gitopper-ctl freeze <service-name> [<duration>] @machine
gitopper-ctl unfreeze <service-name> @machine
gitopper-ctl rollback <service-name> <hash> @machine

## also a state?
gitopper-ctl redo <service-name> @machine    # delete repo, and refetch - impact on service?

TEXT OUTPUT? So not really, or json? Never know if you have everything???

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

## TODO

* Use https://cli.urfave.org/v2/getting-started/ for command cli handling
* Use gorilla mux to do the webserving side
* Write client - to check webserving side.
* Metrics
