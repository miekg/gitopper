Blah (name TBD) is an experiment to see if https://miek.nl/2022/november/15/provisioning-services/
is actually a sane way of doing thing.

The mind says 'yes', reality says '...' ?



## Notes

## metrics

gitopper_service_hash{hash="...", service="...", machine="...."} 1

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

bind mounts are added on top of each other

### config

~~~
grafana.atoom.net {
        package grafana
        action reload grafana-server
        repo {
            url git@gitlab.com/sys/data
            mountpoint /mnt/grafana
        }
        # define the directories for the sparse checkout and how to bind mount them
        /etc/grafana -> grafana/etc
        /var/lib/grafana/dashboards -> grafana/dashboards
}
~~~

~~~ txt
{
    # global options
    url git@github.com/miekg/blah-origin
}

grafana.atoom.net {
        package grafana
        action reload grafana-server
        mountpoint /mnt/grafana
        directories grafana

        # define the directories for the sparse checkout and how to bind mount them
        /etc/grafana -> grafana/etc
        /var/lib/grafana/dashboards -> grafana/dashboards
}
~~~

## Code

Do we need plugins for this things? Maybe other type of remotes, like mercurial or something?

## Remote Interface

first metrics, then remote interface

Rest like???

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

TEXT OUTPUT? So not really, or json? Never know if you have everything???
