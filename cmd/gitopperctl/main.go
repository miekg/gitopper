package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/miekg/gitopper/proto"
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
			&cli.BoolFlag{
				Name:  "m",
				Usage: "machine readable output",
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
						Action:  cmdMachines,
					},
					{
						Name:    "service",
						Aliases: []string{"s"},
						Usage:   "list service @machine [<service>]",
						Action:  cmdService,
					},
				},
			},
			{
				Name:    "do",
				Aliases: []string{"d"},
				Usage:   "apply state changes to a service on a machine",
				Subcommands: []*cli.Command{
					{
						Name:    "freeze",
						Aliases: []string{"f"},
						Usage:   "do freeze @machine <service>",
						Action:  cmdFreeze,
					},
					{
						Name:    "unfreeze",
						Aliases: []string{"u"},
						Usage:   "do unfreeze @machine <service>",
						Action:  cmdUnfreeze,
					},
					{
						Name:    "rollback",
						Aliases: []string{"r"},
						Usage:   "do rollback @machine <service> <hash>",
						Action:  cmdRollback,
					},
					{
						Name:    "pull",
						Aliases: []string{"p"},
						Usage:   "do pull @machine <service>",
						Action:  cmdPull,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func cmdPull(ctx *cli.Context) error {
	at, err := atMachine(ctx)
	if err != nil {
		return err
	}
	service := ctx.Args().Get(1)
	if service == "" {
		return fmt.Errorf("need service")
	}
	_, err = querySSH(ctx, at, "/do/pull", service)
	return err
}

func cmdRollback(ctx *cli.Context) error {
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
	_, err = querySSH(ctx, at, "/do/rollback", service, hash)
	return err
}

func cmdUnfreeze(ctx *cli.Context) error {
	at, err := atMachine(ctx)
	if err != nil {
		return err
	}
	service := ctx.Args().Get(1)
	if service == "" {
		return fmt.Errorf("need service")
	}
	_, err = querySSH(ctx, at, "/do/unfreeze", service)
	return err
}

func cmdFreeze(ctx *cli.Context) error {
	at, err := atMachine(ctx)
	if err != nil {
		return err
	}
	service := ctx.Args().Get(1)
	if service == "" {
		return fmt.Errorf("need service")
	}
	_, err = querySSH(ctx, at, "/do/freeze", service)
	return err
}

func tblPrint(writer io.Writer, line []string) {
	fmt.Fprintln(writer, strings.Join(line, "\t"))
}

func cmdService(ctx *cli.Context) error {
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
	if ctx.Bool("m") {
		fmt.Print(string(body))
		return nil
	}
	tbl := new(tabwriter.Writer)
	tbl.Init(os.Stdout, 0, 8, 1, ' ', 0)
	tblPrint(tbl, []string{"#", "SERVICE", "HASH", "STATE", "INFO", "SINCE"})
	for i, ls := range ls.ListServices {
		tblPrint(tbl, []string{strconv.FormatInt(int64(i), 10), ls.Service, ls.Hash, ls.State, ls.StateInfo, ls.StateChange})
	}
	_ = tbl.Flush()
	return nil
}

func cmdMachines(ctx *cli.Context) error {
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
	if ctx.Bool("m") {
		fmt.Print(string(body))
		return nil
	}
	tbl := new(tabwriter.Writer)
	tbl.Init(os.Stdout, 0, 8, 1, ' ', 0)
	tblPrint(tbl, []string{"#", "MACHINE", "ACTUAL"})
	for i, m := range lm.ListMachines {
		tblPrint(tbl, []string{strconv.FormatInt(int64(i), 10), m.Machine, m.Actual})
	}
	_ = tbl.Flush()
	return nil
}
