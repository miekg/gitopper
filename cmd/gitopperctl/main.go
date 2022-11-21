package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/miekg/gitopper/proto"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v2"
	"go.science.ru.nl/log"
)

func atMachine(ctx *cli.Context) (string, error) {
	at := ctx.Args().First()
	if at == "" {
		return "", fmt.Errorf("expected @<machine>")
	}
	if !strings.HasPrefix(at, "@") {
		return "", fmt.Errorf("expected @<machine>")
	}
	return at[1:], nil
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "i",
				Value: "",
				Usage: "identity file",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "list",
				Aliases: []string{"ls", "l"},
				Usage:   "list machines, services or a single service",
				Subcommands: []*cli.Command{
					{
						Name:    "machines",
						Aliases: []string{"m"},
						Usage:   "list machines @machine",
						Action: func(ctx *cli.Context) error {
							at, err := atMachine(ctx)
							if err != nil {
								return err
							}
							body, err := querySSH(ctx, at, "/list/machine")
							if err != nil {
								return err
							}
							lm := proto.ListMachines{}
							if err := json.Unmarshal(body, &lm); err != nil {
								return err
							}
							tbl := table.New("#", "MACHINE", "ACTUAL")
							for i, m := range lm.ListMachines {
								tbl.AddRow(i, m.Machine, m.Actual)
							}
							tbl.Print()
							return nil
						},
					},
					{
						Name:    "service",
						Aliases: []string{"s"},
						Usage:   "list service @machine [<service>]",
						Action: func(ctx *cli.Context) error {
							at, err := atMachine(ctx)
							if err != nil {
								return err
							}
							var body []byte
							service := ctx.Args().Get(1)
							if service != "" {
								body, err = querySSH(ctx, at, "/list/service", service)
							} else {
								body, err = querySSH(ctx, at, "/list/service")
							}
							if err != nil {
								return err
							}
							ls := proto.ListServices{}
							if err := json.Unmarshal(body, &ls); err != nil {
								return err
							}
							tbl := table.New("#", "SERVICE", "HASH", "STATE", "INFO", "SINCE")
							for i, ls := range ls.ListServices {
								tbl.AddRow(i, ls.Service, ls.Hash, ls.State, ls.StateInfo, ls.StateChange)
							}
							tbl.Print()
							return nil
						},
					},
				},
			},
			{
				Name:    "state",
				Aliases: []string{"st", "s"},
				Usage:   "apply state changes to a service on a machine",
				Subcommands: []*cli.Command{
					{
						Name:    "freeze",
						Aliases: []string{"f"},
						Usage:   "state freeze @machine <service>",
						Action: func(ctx *cli.Context) error {
							at, err := atMachine(ctx)
							if err != nil {
								return err
							}
							service := ctx.Args().Get(1)
							if service == "" {
								return fmt.Errorf("need service")
							}
							_, err = querySSH(ctx, at, "/state/freeze", service)
							return err
						},
					},
					{
						Name:    "unfreeze",
						Aliases: []string{"u"},
						Usage:   "state unfreeze @machine <service>",
						Action: func(ctx *cli.Context) error {
							at, err := atMachine(ctx)
							if err != nil {
								return err
							}
							service := ctx.Args().Get(1)
							if service == "" {
								return fmt.Errorf("need service")
							}
							_, err = querySSH(ctx, at, "/state/unfreeze", service)
							return err
						},
					},
					{
						Name:    "rollback",
						Aliases: []string{"r"},
						Usage:   "state rollback @machine <service> <hash>",
						Action: func(ctx *cli.Context) error {
							at, err := atMachine(ctx)
							if err != nil {
								return err
							}
							service := ctx.Args().Get(1)
							if service == "" {
								return fmt.Errorf("need service")
							}
							hash := ctx.Args().Get(2)
							if hash == "" {
								return fmt.Errorf("need hash to rollback to")
							}
							_, err = querySSH(ctx, at, "/state/rollback", service, hash)
							return err
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
