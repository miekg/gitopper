package main

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/gitopper/osutil"
	"github.com/phayes/freeport"
	flag "github.com/spf13/pflag"
)

func TestFlags(t *testing.T) {
	for _, test := range []struct {
		Arguments []string
		Want      ExecContext
	}{
		{
			Arguments: []string(nil),
			Want: ExecContext{
				Hosts:        []string{osutil.Hostname()}, // default value
				ConfigSource: "",
				SAddr:        ":2222",
				MAddr:        ":9222",
				Debug:        false,
				Restart:      false,
				Root:         true,
				Duration:     5 * time.Minute,
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
				Hosts:        []string{"me", "you"}, // only in main() we add our hostname
				ConfigSource: "/dev/null",
				SAddr:        ":3000",
				MAddr:        ":2000",
				Debug:        true,
				Restart:      true,
				Root:         true,
				Duration:     5 * time.Minute,
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
	err := run(&ExecContext{Root: true})
	if got, want := err, ErrNotRoot; !errors.Is(got, want) {
		t.Errorf("run(exec) = %v, want %v", got, want)
	}
}

func port(t *testing.T) int {
	t.Helper()
	p, err := freeport.GetFreePort()
	if err != nil {
		t.Fatalf("acquire port: %v", err)
	}
	return p
}

// httpClient returns a pristine HTTP client that does not use the shared
// connection cache.  Shared connection caches have produced flaky tests in the
// past.
func httpClient() *http.Client {
	return &http.Client{Transport: new(http.Transport)}
}

func TestServeMonitoring(t *testing.T) {
	p := port(t)
	exec := &ExecContext{
		MAddr:   fmt.Sprintf(":%v", p),
		HTTPMux: http.NewServeMux(),
	}
	var controllerWG, workerWG sync.WaitGroup
	workerWG.Add(1)
	if err := serveMonitoring(exec, &controllerWG, &workerWG); err != nil {
		t.Fatalf("serveMonitoring(exec, &controllerWG, &workerWG) = %v, want %v", err, error(nil))
	}
	client := httpClient()
	resp, err := client.Get(fmt.Sprintf("http://localhost:%v/metrics", p))
	if err != nil {
		t.Fatalf(`client.Get("http://localhost:%v/metrics") err = %v, want %v`, p, err, error(nil))
	}
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("after HTTP GET, resp.StatusCode = %v, want %v", got, want)
	}
	t.Log("Stopping workers; should unlock controllerWG")
	workerWG.Done()
	controllerWG.Wait()
}
