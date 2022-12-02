# Gitopper

GitOps for the non-Kubernetes folks.

Watch a remote git repo, pull changes and HUP the service process. For a design doc see:
<https://miek.nl/2022/november/15/provisioning-services/>.

A sparse (but with full history) git checkout will be done, so each service will only see the files
it will actually need. Several bind mounts are then setup to give the service access to the file in
Git. If the target directories don't exist, they will be created, with the current user - if
specified.

This tool does little more than just pull the repo, but the little it brings to the table allows for
a GitOps workflow without resorting to Kubernetes like environments.

The Git repository that you are using to provision the services must have at least one
(sub)directory for each service.

Gitopper will install packages if told to do so. It will not upgrade or downgrade them, assuming
there is a better way of doing those.

## Features Implemented

From the [doc](https://miek.nl/2022/november/15/provisioning-services/):

> * metrics, so rollouts/updates can be tracked;
> * diff detection, so we know state doesn’t reconcile;
> * out of band rollbacks;
> * no client side processing;
> * canarying.

- *Metrics*: are included see below, they export a Git hash, so a rollout can be tracked.
- *Diff detection*: possible using the metrics or gitopperctl.
- *Out of band rollbacks*: use gitopperctl to bypass the normal Git workflow.
- *No client side processing*: files are used as the are in the Git repo.
- *Canarying*: give a service a different branch to check out.

## Quick Start

- Compile the gitopper binary: `go build`
- Generate a toy SSH key: `ssh-keygen -t ed25519` and make it write to an `id_ed25519_gitopper` file.
- Put the path to the *PUBLIC* key (ending in .pub) in the `keys` fields of `[global]` in config.toml
- Start as root: `sudo ./gitopper -c config.toml -h localhost`

And things should work then. I.e. in /etc/prometheus you should see the content of the
*miekg/gitopper-config* repository.

The checked out git repo in /tmp/prometheus should _only_ contain the prometheus directory thanks to
the sparse checkout. Changes made to any other subdirectory in that repo do not trigger a prometheus
reload.

Then with cmd/gitopperctl/gitopperctl you can query the server:

~~~
./gitopperctl -i <path-to-your-key> list service @localhost
#  SERVICE     HASH                                     STATE  INFO  CHANGED
0  prometheus  606eb576c1b91248e4c1c4cd0d720f27ac0deb70 OK           Fri, 18 Nov 2022 09:14:52 UTC
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

If ROLLBACK stays in ROLLBACK for a longer period it means the commit it needs to rollback to can't
be found.

## Config File

~~~ toml
# global options are applied if a service doens't list them
[global]
upstream = "https://github.com/miekg/gitopper-config"  # repository where to download from
mount = "/tmp"                                     # directory where to download to, mount+service is used as path
# ssh keys that are allowed in via authorized keys
keys =[
	{ path = "keys/miek_id_ed25519_gitopper.pub" },
	{ path = "keys/another_key.pub", ro = true },
]

# each managed service has an entry like this
[[services]]
machine = "prometheus"        # hostname of the machine, so a host knows when to pick this up.
branch = "main"               # what branch to check out
service = "prometheus"        # service identifier, if it's used by systemd it must be the systemd service name
package = "prometheus"        # as used by package mgmt, may be empty (not implemented yet)
user = "prometheus"           # do the check out with this user
action = "reload"             # call systemctl <action> <service> when the git repo changes, may be empty
# what directories from the repo to mount under the local directories
dirs = [
    { local = "/etc/prometheus", link = "prometheus/etc" },   # prometheus/etc *in the repo* should be mounted under /etc/prometheus
]
~~~

Note that `machine` above should match either the machine name ($HOSTNAME) or any of the values you
give on the `-h` flag. This allows you to create services that run everywhere, by defining a service
that have name (say) "localhost" and then deploying gitopper with `-h localhost` on every machine.

### How to Break It

Moving to a new user, will break git pull, with an error like 'dubious ownership of reposiory'. If
you want a different owner for a service, it's best to change the mount as well so you get a new
repo. Gitopper is currently not smart enough to detect this and fix things on the fly.

## Interface

Gitopper opens two ports: 9222 for metrics and 2222 for the rest-protocol-over-SSH. For any
interaction with gitopper over this port you're key must be configured for it.

The following services are implemented:

* List all defined machines.
* List services run on the machine.
* List a specific service.
* Freeze a service to the current git commit.
* Unfreeze a service, i.e. to let it pull again.
* Rollback a service to a specific commit.

For each of these the client will execute a "command" and will parse the returned json into a nice
table.

## Metrics

The following metrics are exported:

* gitopper_service_hash{"service"} \<hash\>
* gitopper_service_state{"service"} \<state\>
* gitopper_service_change_timestamp{"service"} \<epoch\>
* gitopper_machine_git_errors_total - total number of errors when running git.
* gitopper_machine_git_ops_total - total number of git runs.

Metrics are available under the /metrics endpoint on port 9222.

## Exit Code

Gitopper has following exit codes:

0 - normal exit
2 - SIGHUP seen (signal to systemd to restart us)

## Bootstrapping

There are a couple of options that allow gitopper to bootstrap itself *and* make gitopper to be
managed by gitopper. Basically those options allow you to specificy a service on the command line.
Gitopper will check out the repo and then proceed to read the config *in that repo* and setup
everything from there.

I.e.:

~~~
... -c config.toml -U https://github.com/miekg/gitopper-config -D gitopper -M /tmp/
~~~

Will sparse check out (only the `gitopper` (-D flag) directory) of the repo *gitopper-config* (-U flag)
in /tmp/gitopper (-M flag, internally '/gitopper' is added) and will then proceed to parse the
config file /tmp/gitopper/gitopper/config.toml and proceed with a normal startup.

Note this setup implies that you *must* place config.toml *inside* a `gitopper` directory, just as
the other services must have their own subdirectories, gitopper needs one too.

The gitopper service self is *also* added to the managed services which you can inspect with
gitopperctl.

Any keys that have *relative* paths, will also be changed to key inside this Git managed directory
and pick up keys *from that repo*.

The `-P` flag can be given to pull the repository even if it already exists, sometimes you need to
the newest version to properly bootstrap. For normal services the "git pull" routine will
automatically rectify it and restart the service.

## Client

A client is included in cmd/gitopperctl. It has its own README.md.

## Authentication

Authentication uses SSH, so it fits in with the rest of the infrastructure.
