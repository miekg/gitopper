# gitopperctl

Small cli to inspect control gitopper remotely. The command line syntax follow other \*-ctl tools a
bit.

The two main branches of use are `list` and `state`.

~~~
./gitopperctl list machine @<host>
./gitopperctl list service @<host>
./gitopperctl list service  @<host> <service>
~~~

In order:

1. List all machines defined in the config file for gitopper running on `<host>`.
2. List all services that are controlled on `<host>`.
3. List a specific service on `<host>`.

Each will output a simple table with the information:

~~~
% ./gitopperctl list service @localhost grafana-server
SERVICE         HASH                                      STATE  INFO  SINCE
grafana-server  606eb576c1b91248e4c1c4cd0d720f27ac0deb70  OK           2022-11-18 13:29:44.824004812 +0000 UTC
~~~

`--help` to show implemented subcommands.

## Manipulating Services

Freezing (make it stop updating to the latest commit), until a unfreeze:

~~~
./gitopperctl state freeze   @<host> <service>
./gitopperctl state unfreeze @<host> <service>
~~~

Rolling back to a previous commit, hash needs to be full length:

~~~
./gitopperctl state rollback @<host> <service> <hash>
~~~

And this can be abbreviated to:

~~~
./gitopperctl s r @<host> <service> <hash>
~~~

## Example

This is a small example of this tool interacting with the daemon.

- check current service

~~~
% ./gitopperctl list service @localhost grafana-server
SERVICE         HASH                                      STATE  INFO  SINCE
grafana-server  606eb576c1b91248e4c1c4cd0d720f27ac0deb70  OK           0001-01-01 00:00:00 +0000 UTC
~~~

-  rollback

~~~
% ./gitopperctl state rollback @localhost grafana-server 8df1b3db679253ba501d594de285cc3e9ed308ed
~~~

- check
~~~
% ./gitopperctl list service @localhost grafana-server
SERVICE         HASH                                      STATE     INFO                                      SINCE
grafana-server  606eb576c1b91248e4c1c4cd0d720f27ac0deb70  ROLLBACK  8df1b3db679253ba501d594de285cc3e9ed308ed  2022-11-18 13:28:42.619731556 +0000 UTC
~~~

- check state, rollback done. Now state is FREEZE

~~~
% ./gitopperctl list service @localhost grafana-server
SERVICE         HASH                                      STATE   INFO                                                      SINCE
grafana-server  8df1b3db679253ba501d594de285cc3e9ed308ed  FREEZE  ROLLBACK: 8df1b3db679253ba501d594de285cc3e9ed308ed  2022-11-18 13:29:17.92401403 +0000 UTC
~~~

- unfreeze and let it pick up changes again

~~~
% ./gitopperctl state unfreeze @localhost grafana-server
~~~

- check the service

~~~
% ./gitopperctl list service @localhost grafana-server
SERVICE         HASH                                      STATE  INFO  SINCE
grafana-server  8df1b3db679253ba501d594de285cc3e9ed308ed  OK           2022-11-18 13:29:44.824004812 +0000 UTC
~~~

- and updated to new hash

~~~
% ./gitopperctl list service @localhost grafana-server
SERVICE         HASH                                      STATE  INFO  SINCE
grafana-server  606eb576c1b91248e4c1c4cd0d720f27ac0deb70  OK           2022-11-18 13:29:44.824004812 +0000 UTC
~~~
