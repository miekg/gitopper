//go:build root

package main

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestBindMounts(t *testing.T) {
	local, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	link, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	s := &Service{
		User: "daemon",
		Dirs: []Dir{{
			Local: local,
			Link:  link,
		}},
	}

	mounts, err := s.bindmount()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		ctx := context.TODO()
		cmd := exec.CommandContext(ctx, "umount", s.Dirs[0].Link)
		t.Logf("running %v", cmd.Args)
		cmd.Run()
	}()

	if mounts != 1 {
		t.Fatalf("expected %d mounts, got %d", 1, mounts)
	}
}
