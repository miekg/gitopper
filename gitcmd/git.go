// Package gitcmd has a bunch of convience functions to work with Git.
// Each machine should use it's own Git.
package gitcmd

import (
	"context"
	"os"
	"os/exec"
	"path"

	"github.com/miekg/gitopper/log"
)

type Git struct {
	url   string
	mount string
	dirs  []string

	cwd string
}

// New returns a pointer to an intialized Git.
func New(url, mount string, dirs []string) *Git {
	g := &Git{
		url:   url,
		mount: mount,
		dirs:  dirs,
	}
	return g
}

func (g *Git) run(args ...string) error {
	log.Infof("running %v", args)
	ctx := context.TODO()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.cwd
	log.Debugf("DIR", cmd.Dir)

	out, err := cmd.CombinedOutput()
	log.Debug(string(out))

	return err
}

// IsCheckedOut will check g.mount and if it has an .git sub directory we assume the checkout has been done.
func (g *Git) IsCheckedOut() bool {
	info, err := os.Stat(path.Join(g.mount, ".git"))
	if err != nil {
		return false
	}
	return info.Name() == ".git" && info.IsDir()
}

// Checkout will do the initial check of the git repo. If the g.mount directory already exist and has
// a .git subdirectory, it will assume the checkout has been done during a previuos run.
func (g *Git) Checkout() error {
	if g.IsCheckedOut() {
		return nil
	}

	g.cwd = ""
	err := g.run("clone", "--filter=blob:none", "--no-checkout", "--sparse", g.url, g.mount)
	if err != nil {
		return err
	}
	g.cwd = g.mount
	defer func() { g.cwd = "" }()

	args := []string{"sparse-checkout", "add"}
	args = append(args, g.dirs...)
	err = g.run(args...)

	return err
}

// Pull pulls from upstream.
func (g *Git) Pull() error {
	g.cwd = g.mount
	defer func() { g.cwd = "" }()

	err := g.run("pull")

	return err
}

// Diff detect if a pull has updated any files, the returned boolean is true in that case.
func (g *Git) Diff() (bool, error) {
	g.cwd = g.mount
	defer func() { g.cwd = "" }()

	args := []string{"diff", "HEAD", "HEAD^", "--"}
	args = append(args, g.dirs...) // can we check multiple dirs?
	err := g.run(args...)
	if err != nil {
		return false, err
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode() == 0, nil
	}

	return false, nil
}
