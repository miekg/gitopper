%%%
title = "gitopper 8"
area = "System Administration"
workgroup = "Git Operations"
%%%

gitopper
=====

## Name

gitopper - watch a git repository, pull changes and reload the server process

## Synopsis

`gitopper [OPTION]...` `-c` **CONFIG**

## Description

Gitopper is GitOps for non-Kubernetes folks it watches a remote git repo, pulls changes and HUP the
server (service) process.

A sparse (but with full history) git checkout will be done, so each service will only see the files
it will actually need. Several bind mounts are then setup to give the service access to the file(s)
in Git. If the target directories don't exist, they will be created, with the current user - if
specified.

This tool does little more than just pull the repo, but the little it brings to the table allows for
a GitOps workflow without resorting to Kubernetes like environments.

The Git repository that you are using to provision the services must have at least one
(sub)directory for each service.

Gitopper will install packages if told to do so. It will not upgrade or downgrade them, assuming
there is a better way of doing those.

The remote interface of gitopper uses SSH keys for authentication, this hopefully helps to fit in,
in a sysadmin organisation.

The following features are implemented:

- *Metrics*: are included see below, they export a Git hash, so a rollout can be tracked.
- *Diff detection*: possible using the metrics or gitopperctl.
- *Out of band rollbacks*: use gitopperctl to bypass the normal Git workflow.
- *No client side processing*: files are used as they are in the Git repo.
- *Canarying*: give a service a different branch to check out.

The options are:

**-h, --hosts strings**
:  hosts (comma separated) to impersonate, local hostname is always added

**-c, --config string**
:  config file to read

**-s, --ssh string**
:  ssh address to listen on (default ":2222")

**-m, --metric string**
:  http metrics address to listen on (default ":9222")

**-d, --debug**
:  enable debug logging

**-r, --restart**
:   send SIGHUP to ourselves when config changes

**-o, --root**
:  require root permission, setting to false can aid in debugging (default true)

**-t, --duration duration**
:  default duration between pulls (default 5m0s)

For bootstrapping gitopper itself the following options are available:

**-U, --upstream string**
:  use this git repo to clone and to bootstrap from

**-D, --directory string**
:  directory to sparse checkout (default "gitopper")

**-B, --branch string**
:   check out in this branch (default "main")

**-M, --mount string**
:   check out into this directory, -c is relative to this directory

**-P, --pull**
:   pull (update) the git repo to the newest version before starting

## Quick Start

- Generate a toy SSH key: `ssh-keygen -t ed25519` and make it write to an `id_ed25519_gitopper` file.
- Put the path to the *PUBLIC* key (ending in .pub) in the `keys` fields of `[global]` of the
  following config.toml file.
- Start as root: `sudo ./gitopper -c config.toml -h localhost`

~~~ toml
[global]
upstream = "https://github.com/miekg/gitopper-config"
branch = "main"
mount = "/tmp/gitopper"
keys = [
	{ path = "keys/id_ed25519_gitopper.pub", ro = true },
]

[[services]]
machine = "localhost"
service = "prometheus"
user = "prometheus"
package = "prometheus"
action = "reload"
dirs = [
	{ local = "/etc/prometheus", link = "prometheus/etc" },
]
~~~

And things should work then. I.e. in /etc/prometheus you should see the content of the
*miekg/gitopper-config* repository. Note that the prometheus package is installed, because `package`
is mentioned in the config file.

The checked out git repo in /tmp/prometheus should _only_ contain the prometheus directory thanks to
the sparse checkout. Changes made to any other subdirectory in that repo do not trigger a prometheus
reload.

Then with gitopperctl(8) you can query the server:

~~~
./gitopperctl -i <path-to-your-key> list service @localhost
#  SERVICE     HASH       STATE  INFO  CHANGED
0  prometheus  606eb576   OK           Fri, 18 Nov 2022 09:14:52 UTC
~~~

## Services

A service can be in 5 states: OK, FREEZE, ROLLBACK (which is a FREEZE to a previous commit) and
BROKEN/DIFF.

These states are not carried over when gitopper crashes/stops (maybe we want this to be persistent,
would be nice to have this state in the git repo somehow?).

* `OK`: everything is running and we're tracking upstream.
* `FREEZE`: everything is running, but we're not tracking upstream.
* `ROLLBACK`: everything is running, but we're not tracking upstream *and* we're pinned to an older
  commit. This state is quickly followed by FREEZE if we were successful rolling back, otherwise
  BROKEN (systemd error)  of DIFF (git error)
* `BROKEN`: something with the service is broken, we're still tracking upstream. I.e. systemd error.
* `DIFF`: the git repository can't be reconciled with upstream. I.e. git error.

ROLLBACK is a transient state and quickly moves to FREEZE, unless something goes wrong then it
becomes BROKEN, or DIFF depending on what goes wrong (systemd, or git respectively).

~~~
 +-------------------------+
 |                         |
 v                         |
*OK -------> ROLLBACK ---> FREEZE
 |          /    \         |
 |         /      \        v
 |        |        |       |
 |        v        v       |
 |      BROKEN     DIFF    |
 |        |         |      |
 |        |         |      |
 +--------+---------+------+
~~~

- `*OK` is the start state
- from `OK` and `FREEZE` we can still end up in `BROKEN` and `FREEZE` and vice versa.

## Config File

~~~ toml
# global options are applied if a service doesn't list them
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
service = "prometheus"        # service identifier, if it's used by systemd it must be the systemd service name
action = "reload"             # call systemctl <action> <service> when the git repo changes, may be empty
branch = "main"               # what branch to check out
package = "prometheus"        # as used by package mgmt, may be empty (not implemented yet)
user = "prometheus"           # do the check out with this user
# what directories or files from the repo to mount under the local directories
dirs = [
    { local = "/etc/prometheus", link = "prometheus/etc" },   # prometheus/etc *in the repo* should be mounted under /etc/prometheus
    { local = "/etc/caddy/Caddyfile", link = "caddy/etc/Caddyfile", file = true },   # caddy/etc/Caddyfile *in the repo* should be mounted under /etc/caddy/Caddyfile
]
~~~

Note that `machine` above should match either the machine name ($HOSTNAME) or any of the values you
give on the `-h` flag. This allows you to create services that run everywhere, by defining a service
that have name (say) "localhost" and then deploying gitopper with `-h localhost` on every machine.

Options for each service:

- `machine`: the machine where this service should be active. By default `gitopper` will know the
   current hostname, but multiple aliases may be given to it via the `-h` flag.
- `service`: what systemd unit file is used to call `action` on. If service contains an `@` a
  service template unit is assumed and gitopper will then run `systemctl enable <service>` to enable
  the service template.
- `action`: action to use when calling `systemctl <service>`. If empty no systemd command will be
  issued when the repo changes.
- `branch`: what branch to use in the checked out repo. Note different branches that use the *same*
  repository on disk, will error on startup.
- `package`: what package to install for this service. If empty, no package will be installed.
- `user`: what user should the git repository belong to.
- `dirs`: describe the mapping between directories and files in the repository and on the local
  disk. `local` is the *on disk* name, and `link` is the *relative* path of the directory or file in
  the git repo. If a single file is used, `file` should be set to true.

### How to Break It

Moving to a new user, will break git pull, with an error like 'dubious ownership of repository'. If
you want a different owner for a service, it's best to change the mount as well so you get a new
repo. Gitopper is currently not smart enough to detect this and fix things on the fly.

## Interface

Gitopper opens two ports: 9222 for metrics and 2222 for the rest-protocol-over-SSH. For any
interaction with gitopper over this port your key must be configured for it.

The following services are implemented:

* List all defined machines.
* List services run on the machine.
* List a specific service.
* Freeze a service to the current git commit.
* Unfreeze a service, i.e. to let it pull again.
* Rollback a service to a specific commit.

For each of these gitopperctl(8) will execute a "command" and will parse the returned JSON into a nice
table.

## Metrics

The following metrics are exported:

* gitopper_service_state{"service"} \<state\>
* gitopper_service_change_time_seconds{"service"} \<epoch\>
* gitopper_machine_git_errors_total - total number of errors when running git.
* gitopper_machine_git_ops_total - total number of git runs.

Metrics are available under the /metrics endpoint on port 9222.

## Exit Code

Gitopper has following exit codes:

0 - normal exit
2 - SIGHUP seen (signal to systemd to restart us)

## Bootstrapping

There are a couple of options that allow gitopper to bootstrap itself *and* make gitopper to be
managed by gitopper. Basically those options allow you to specify a service on the command line.
Gitopper will check out the repo and then proceed to read the config *in that repo* and setup
everything from there.

I.e.:

~~~
... -c config.toml -U https://github.com/miekg/gitopper-config -D gitopper -M /tmp/
~~~

Will sparse check out (only the `gitopper` (-D flag) directory) of the repo *gitopper-config* (-U
flag) in /tmp/gitopper (-M flag, internally '/gitopper' is added) and will then proceed to parse the
config file /tmp/gitopper/gitopper/config.toml and proceed with a normal startup.

Note this setup implies that you *must* place config.toml *inside* a `gitopper` directory, just as
the other services must have their own subdirectories, gitopper needs one too.

The gitopper service self is *also* added to the managed services which you can inspect with
gitopperctl(8).

Any keys that have *relative* paths, will also be changed to key inside this Git managed directory
and pick up keys *from that repo*.

The `-P` flag can be given to pull the repository even if it already exists, sometimes you need to
the newest version to properly bootstrap. For normal services the "git pull" routine will
automatically rectify it and restart the service.

## See Also

See [this design doc](https://miek.nl/2022/november/15/provisioning-services/), and gitopperctl(8).
