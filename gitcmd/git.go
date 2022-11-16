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
	upstream string
	mount    string
	dirs     []string

	cwd string
}

// New returns a pointer to an intialized Git.
func New(upstream, mount string, dirs []string) *Git {
	g := &Git{
		upstream: upstream,
		mount:    mount,
		dirs:     dirs,
	}
	return g
}

func (g *Git) run(args ...string) ([]byte, error) {
	ctx := context.TODO()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.cwd
	log.Infof("running in %q %v", cmd.Dir, cmd.Args)

	out, err := cmd.CombinedOutput()
	log.Debug(string(out))

	return out, err
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
	_, err := g.run("clone", "--filter=blob:none", "--no-checkout", "--sparse", g.upstream, g.mount)
	if err != nil {
		return err
	}

	g.cwd = g.mount
	defer func() { g.cwd = "" }()
	args := []string{"sparse-checkout", "set"}
	args = append(args, g.dirs...)
	_, err = g.run(args...)
	if err != nil {
		return err
	}

	_, err = g.run("checkout")
	return err
}

// Pull pulls from upstream.
func (g *Git) Pull() error {
	g.cwd = g.mount
	defer func() { g.cwd = "" }()

	_, err := g.run("pull")
	return err
}

// Diff detect if a pull has updated any files, the returned boolean is true in that case.
func (g *Git) Diff() (bool, error) {
	g.cwd = g.mount
	defer func() { g.cwd = "" }()

	args := []string{"diff", "HEAD", "HEAD^", "--"}
	args = append(args, g.dirs...) // can we check multiple dirs?
	_, err := g.run(args...)
	if err != nil {
		return false, err
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode() == 0, nil
	}

	return false, nil
}

// Hash returns the git hash of HEAD in the repo in g.mount
func (g *Git) Hash() (string, error) {
	g.cwd = g.mount
	defer func() { g.cwd = "" }()

	out, err := g.run("rev-parse", "HEAD")
	if err != nil {
		return "", err
	}

	hash := string(out)
	if len(hash) >= 41 {
		hash = hash[:40]
	}

	return hash, nil
}

func (g *Git) Repo() string { return g.mount }
