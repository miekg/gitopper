// Package gitcmd has a bunch of convience functions to work with Git.
// Each machine should use it's own Git.
package gitcmd

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
	"strings"
	"syscall"

	"go.science.ru.nl/log"
)

type Git struct {
	upstream string
	mount    string
	dirs     []string
	user     string

	cwd string
}

// New returns a pointer to an intialized Git.
func New(upstream, mount, user string, dirs []string) *Git {
	g := &Git{
		upstream: upstream,
		mount:    mount,
		dirs:     dirs,
		user:     user,
	}
	return g
}

func (g *Git) run(args ...string) ([]byte, error) {
	ctx := context.TODO()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.cwd
	if g.user != "" {
		uid, gid := userLookup(g.user)
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	}

	log.Infof("running in %q as %q %v", cmd.Dir, g.user, cmd.Args)

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

// Pull pulls from upstream. If the returned bool is true there were updates.
func (g *Git) Pull() (bool, error) {
	g.cwd = g.mount
	defer func() { g.cwd = "" }()

	out, err := g.run("pull", "--stat")
	if err != nil {
		return false, err
	}
	return g.OfInterest(out), nil
}

// Hash returns the git hash of HEAD in the repo in g.mount. Empty string is returned in case of an error.
func (g *Git) Hash() string {
	g.cwd = g.mount
	defer func() { g.cwd = "" }()

	out, err := g.run("rev-parse", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (g *Git) Repo() string { return g.mount }

func userLookup(u string) (int64, int64) {
	u1, err := user.Lookup(u)
	if err != nil {
		return 0, 0
	}
	uid, _ := strconv.ParseInt(u1.Uid, 10, 32)
	gid, _ := strconv.ParseInt(u1.Gid, 10, 32)
	return uid, gid
}
