package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/miekg/gitopper/gitcmd"
	"github.com/urfave/cli/v2"
	"go.science.ru.nl/log"
)

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "p",
				Value: "#",
				Usage: "comment prefix to use",
			},
			&cli.StringFlag{
				Name:  "l",
				Value: "en",
				Usage: "language to use",
			},
		},
		Name:  "hdr",
		Usage: "print a header suitable for inclusion in a file",
		Action: func(ctx *cli.Context) error {
			file := ctx.Args().Get(0)
			if file == "" {
				log.Fatal("Want a file argument")
			}
			gc := gitcmd.New("", "", "", "", nil) // don't need any of these.
			url := gc.OriginURL()
			if url == "" {
				log.Fatal("Failed to get upstream origin URL")
			}
			branch := gc.BranchCurrent()
			if branch == "" {
				log.Fatal("Failed to get current branch")
			}
			relpath := gc.LsFile(file)
			if relpath == "" {
				log.Warningf("Failed to get relative path for: %q, omitting file path from output", file)
			}
			comment := ctx.String("p")
			headerfmt, ok := Header[ctx.String("l")]
			if !ok {
				headerfmt = Header["en"]
			}
			fmt.Printf(headerfmt, comment, comment, transformURL(ctx, url, relpath, branch))
			fmt.Println()
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// transformURL will transform a git@ url to a https:// one, taking into gitlab vs github into account. The returns
// string is a valid URL that points to the file in relpath.
func transformURL(ctx *cli.Context, url, relpath, branch string) string {
	// github: https://github.com/miekg/gitopper/blob/main/proto/proto.go
	// gitlab: https://gitlab.science.ru.nl/cncz/go/-/blob/main/cmd/graaf/graaf.go
	//
	// git@github.com:miekg/gitopper.git will be transformed to:
	// https://github.com/miekg/gitopper

	if strings.HasPrefix(url, "git@") {
		if strings.HasSuffix(url, ".git") {
			url = url[:len(url)-len(".git")]
		}
		url = url[len("git@"):]
		url = strings.Replace(url, ":", "/", 1)
		url = "https://" + url
	}
	if relpath == "" {
		return url
	}
	// url has been normalized, add path
	sep := "/-/blob/" + branch + "/"
	if strings.Contains(url, "//github.com/") {
		sep = "/blob/" + branch + "/"
	}
	return url + sep + relpath
}

var Header = map[string]string{
	"en": "%s Do not edit this file, it's managed by `gitopper'. The canonical source can be found at:\n%s   %s",
	"nl": "%s Bewerk dit bestand niet, het wordt beheerd door `gitopper'. De canonieke bron is te vinden op:\n%s   %s",
}
