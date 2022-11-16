// Package gitcmd has a bunch of convience functions to work with Git.
// Each machine should use it's own Git.
package gitcmd

import (
	"context"
	"os/exec"
)

type Git struct {
	// other stuff??
	url   string
	mount string
	dirs  []string

	cwd string
}

// New returns a pointer to an intialized Git.
func New(url string) *Git {
	g := &Git{
		url: url,
	}
	return g
}

func (g *Git) run(args ...string) error {
	ctx := context.TODO()
	cmd := exec.CommandContext(ctx, "git", args...)
	if g.cwd != "" {
		cmd.Dir = g.cwd
	}

	return cmd.Run()
}

func (g *Git) Checkout(dirs []string) error {
	err := g.run("clone", "--filter=blob:none", "--no-checkout", "--sparse", g.url, g.mount)
	if err != nil {
		return err
	}
	g.cwd = g.mount
	defer func() { g.cwd = "" }()

	args := append([]string{"sparse-checkout"}, "add")
	args = append(args, dirs...)
	err = g.run(args...)
	if err != nil {
		return err
	}

	return nil
}

// Pull pulls from upstream.
func (g *Git) Pull() error {
	g.cwd = g.mount
	defer func() { g.cwd = "" }()

	err := g.run("pull")
	if err != nil {
		return err
	}

	return nil
}

// Diff detect if a pull has updated any files.
func (g *Git) Diff() (bool, error) {
	return false, nil
}
