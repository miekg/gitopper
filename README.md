Blah (name TBD) is an experiment to see if https://miek.nl/2022/november/15/provisioning-services/
is actually a sane way of doing thing.

The mind says 'yes', reality says '...' ?

## Notes

## metrics

~~~ txt
gitopper_machine_info{machine="..."} 1.0
gitopper_service_hash{service="...", hash="..."} 1.0
gitopper_service_frozen{service="..."} 1.0
gitopper_service_ok{service="..."} 1.0
gitopper_service_failure_count{} 1.0
~~~

## Features

Noop feature

## depth

-depth 1 is useless;
we want all systems have the same history, and if the repo becomes big *all* clients will see that
pressure at the same time, not in the ordering in which they came up and received their first pull.

## first commit

There can be 'no commit' which is ok.

~~~
git df HEAD^ HEAD -- grafana
fatal: bad revision 'HEAD^'
~~~

## wipe a repo

We may want to wipe a repo and let the automation reclone in an emergency.

### config

## Code

Do we need plugins for this things? Maybe other type of remotes, like mercurial or something?

## Remote Interface

- list all machines - from the config file?
- list all services from a machine - should be from the config as well...
- then for a service:
    * freeze ?duration=50
    * rollback ?to=<hash> (can error); rollback also freezes, rollback isn't a permanent state.
    * unfreeze
- get status of services, check also upstream repo

curl essentially

basic auth, then system auth... account exists... and in special group??

-ctl is basically a curl client, but make a client, simple as hell

gitopper-ctl list machines
gitopper-ctl list services @machine

gitopper-ctl status <service-name> @machine  git, pull, last update??

gitopper-ctl freeze <service-name> [<duration>] @machine
gitopper-ctl unfreeze <service-name> @machine
gitopper-ctl rollback <service-name> <hash> @machine
gitopper-ctl redo <service-name> @machine    # delete repo, and refetch - impact on service?

TEXT OUTPUT? So not really, or json? Never know if you have everything???
