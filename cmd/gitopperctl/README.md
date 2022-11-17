# gitopperctl

Small cli to inspect (and soon) control gitopper.

~~~
./gitopperctl list machines @<host>
./gitopperctl list services @<host>
./gitopperctl list service  @<host> <service>
~~~

In order:

1. List all machines defined in the config file for gitopper running on `<host>`.
2. List all services that are controlled on `<host>`.
3. List a specific service on `<host>`.

Each will output a simple table with the information:

~~~
% ./gitopperctl list service @localhost grafana-server
SERVICE         HASH     STATE
grafana-server  606eb57  OK
~~~

`--help` show implemented subcommands.

## Manipulating Services

(Not implemented yet)

~~~
./gitopperctl freeze   server @<host> <service>
./gitopperctl unfreeze server @<host> <service>
~~~

rollback

~~~
./gitopperctl freeze   server @<host> <service>

~~~
