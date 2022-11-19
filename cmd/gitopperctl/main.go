package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/miekg/gitopper/proto"
	"github.com/rodaine/table"
	"github.com/urfave/cli/v2"
	"go.science.ru.nl/log"
)

/*
   fmt.Printf("Body : %s", body)
*/

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

func query(at, method string, args ...string) (body []byte, err error) {
	c := http.Client{Timeout: time.Duration(1) * time.Second}
	url := "http://" + at + ":8000/" + strings.Join(args, "/")
	var resp *http.Response
	switch method {
	case "GET":
		resp, err = c.Get(url)
	case "POST":
		resp, err = c.Post(url, "", nil)
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func main() {
	app := &cli.App{
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
							body, err := query(at, "GET", "list", "machines")
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
						Name:  "services",
						Usage: "list services @machine",
						Action: func(ctx *cli.Context) error {
							at, err := atMachine(ctx)
							if err != nil {
								return err
							}
							body, err := query(at, "GET", "list", "services")
							if err != nil {
								return err
							}
							ls := proto.ListServices{}
							if err := json.Unmarshal(body, &ls); err != nil {
								return err
							}
							tbl := table.New("#", "SERVICE", "HASH", "STATE", "INFO", "SINCE")
							for i, ls := range ls.ListServices {
								tbl.AddRow(i, ls.Service, ls.Hash, ls.State, ls.StateInfo, timeIsZero(ls.StateChange))
							}
							tbl.Print()
							return nil
						},
					},
					{
						Name:    "service",
						Aliases: []string{"s"},
						Usage:   "list service @machine <service>",
						Action: func(ctx *cli.Context) error {
							at, err := atMachine(ctx)
							if err != nil {
								return err
							}
							service := ctx.Args().Get(1)
							if service == "" {
								return fmt.Errorf("need service")
							}
							body, err := query(at, "GET", "list", "service", service)
							if err != nil {
								return err
							}
							ls := proto.ListService{}
							if err := json.Unmarshal(body, &ls); err != nil {
								return err
							}
							tbl := table.New("SERVICE", "HASH", "STATE", "INFO", "SINCE")
							tbl.AddRow(ls.Service, ls.Hash, ls.State, ls.StateInfo, timeIsZero(ls.StateChange))
							tbl.Print()
							return nil
						},
					},
				},
			},
			{
				Name:    "state",
				Aliases: []string{"st"},
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
							_, err = query(at, "POST", "state", "freeze", service)
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
							_, err = query(at, "POST", "state", "unfreeze", service)
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
							_, err = query(at, "POST", "state", "rollback", service, hash)
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

// If the string s is a IsZero() time, we return N/A as we don't know when the last state change was.
func timeIsZero(s string) string {
	return s
	t, err := time.Parse(time.RFC1123, s)
	if err != nil {
		return "N/A"
	}
	if t.IsZero() {
		return "N/A"
	}
	return s
}
