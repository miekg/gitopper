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
		// 	resp, err = c.Post(url) more stuff
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
							tbl := table.New("ID", "MACHINE")
							for i, m := range lm.Machines {
								tbl.AddRow(i, m)

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
							tbl := table.New("ID", "SERVICE")
							for i, s := range ls.Services {
								tbl.AddRow(i, s)

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
							tbl := table.New("SERVICE", "HASH", "STATE")
							tbl.AddRow(ls.Service, ls.Hash, ls.State)
							tbl.Print()
							return nil
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
