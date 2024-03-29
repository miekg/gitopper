%%%
title = "gitopperctl 8"
area = "System Administration"
workgroup = "Git Operations"
%%%

gitopperctl
=====

## Name

gitopperctl - interact remotely with gitopper

## Synopsis

`gitopperctl [OPTION]...` *commands* *@host*

## Description

Gitopperctl is an utility to inspect and control gitopper remotely. The command line syntax follow other \*-ctl tools a
bit.

There are only a few options:

**-i value**
: identity file to use for SSH, this flag is mandatory

**-m**
: machine readable output (default: false), output JSON

**--help, -h**
:  show help

The two main branches of use are `list` and `do` **commands**. Note the `-i <sshkey>` argument is
only shown once in these examples:

~~~
./gitopperctl -i ~/.ssh/id_ed25519_gitopper list machine @<host>
./gitopperctl list service @<host>
./gitopperctl list service  @<host> <service>
~~~

In order:

1. List all machines defined in the config file for gitopper running on `<host>`.
2. List all services that are controlled on `<host>`.
3. List a specific service on `<host>`.

Each will output a simple table with the information:

~~~
./gitopperctl list service @localhost grafana-server
SERVICE         HASH      STATE  INFO  SINCE
grafana-server  606eb576  OK           2022-11-18 13:29:44.824004812 +0000 UTC
~~~

Use `--help` to show implemented subcommands.

### Manipulating Services

Freezing (make it stop updating to the latest commit), until a unfreeze:

~~~
./gitopperctl do freeze   @<host> <service>
./gitopperctl do unfreeze @<host> <service>
~~~

Rolling back to a previous commit, hash needs to be a valid hexadecimal value (meaning it must be of
even length):

~~~
./gitopperctl do rollback @<host> <service> <hash>
~~~

And this can be abbreviated to:

~~~
./gitopperctl d r @<host> <service> <hash>
~~~

Or make it to pull now and now wait for the default wait duration to expire:

~~~
./gitopper do pull @<host> <service>
~~~

## Example

This is a small example of this tool interacting with the daemon.

- check current service

~~~
./gitopperctl list service @localhost grafana-server
SERVICE         HASH      STATE  INFO  SINCE
grafana-server  606eb576  OK           0001-01-01 00:00:00 +0000 UTC
~~~

-  rollback

~~~
./gitopperctl do rollback @localhost grafana-server 8df1b3db679253ba501d594de285cc3e9ed308ed
~~~

- check
~~~
./gitopperctl list service @localhost grafana-server
SERVICE         HASH      STATE     INFO                                      SINCE
grafana-server  606eb576  ROLLBACK  8df1b3db679253ba501d594de285cc3e9ed308ed  2022-11-18 13:28:42.619731556 +0000 UTC
~~~

- check do, rollback done. Now state is FREEZE

~~~
./gitopperctl list service @localhost grafana-server
SERVICE         HASH      STATE   INFO                                                      SINCE
grafana-server  8df1b3db  FREEZE  ROLLBACK: 8df1b3db679253ba501d594de285cc3e9ed308ed  2022-11-18 13:29:17.92401403 +0000 UTC
~~~

- unfreeze and let it pick up changes again

~~~
./gitopperctl do unfreeze @localhost grafana-server
~~~

- check the service

~~~
./gitopperctl list service @localhost grafana-server
SERVICE         HASH      STATE  INFO  SINCE
grafana-server  8df1b3db  OK           2022-11-18 13:29:44.824004812 +0000 UTC
~~~

- and updated to new hash

~~~
./gitopperctl list service @localhost grafana-server
SERVICE         HASH      STATE  INFO  SINCE
grafana-server  606eb576  OK           2022-11-18 13:29:44.824004812 +0000 UTC
~~~
