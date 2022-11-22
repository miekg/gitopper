package main

import (
	"errors"
	"flag"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFlags(t *testing.T) {
	for _, test := range []struct {
		Arguments []string
		Want      ExecContext
	}{
		{
			Arguments: []string(nil),
			Want: ExecContext{
				Hosts:        nil,
				ConfigSource: "",
				SAddr:        ":2222",
				MAddr:        ":9222",
				Debug:        false,
				Restart:      false,
				Upstream:     "",
				Dir:          "gitopper",
				Branch:       "main",
				Mount:        "",
			},
		},
		{
			Arguments: []string{
				"-h=me,you",
				"-c=/dev/null",
				"-s=:3000",
				"-m=:2000",
				"-d",
				"-r",
				"-U=/upstream",
				"-D=/sparse",
				"-B=branch",
				"-M=checkout",
			},
			Want: ExecContext{
				Hosts:        []string{"me", "you"},
				ConfigSource: "/dev/null",
				SAddr:        ":3000",
				MAddr:        ":2000",
				Debug:        true,
				Restart:      true,
				Upstream:     "/upstream",
				Dir:          "/sparse",
				Branch:       "branch",
				Mount:        "checkout",
			},
		},
	} {
		fs := flag.NewFlagSet("", flag.ContinueOnError)
		var exec ExecContext
		exec.RegisterFlags(fs)
		if err := fs.Parse(test.Arguments); err != nil {
			t.Fatalf("fs.Parse(%v) = %v, want %v", test.Arguments, err, error(nil))
		}
		if diff := cmp.Diff(exec, test.Want); diff != "" {
			t.Errorf("after parsing %v, exec = %v, want %v\n\ndiff:\n\n%v", test.Arguments, exec, test.Want, diff)
		}
	}
}

func TestEndToEnd(t *testing.T) {
	// TODO: Make generally testable.
	err := run(new(ExecContext))
	if got, want := err, ErrNotRoot; !errors.Is(got, want) {
		t.Errorf("run(exec) = %v, want %v", got, want)
	}
}
