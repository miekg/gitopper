// Package gitcmd has a bunch of convience functions to work with Git.
// Each machine should use it's own Git.
package gitcmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/miekg/gitopper/osutil"
	"go.science.ru.nl/log"
)

type Git struct {
	upstream string
	branch   string
	mount    string
	dirs     []string
	user     string

	cwd string
}

// New returns a pointer to an intialized Git.
func New(upstream, branch, mount, user string, dirs []string) *Git {
	// Git is starting to look a lot like Service....
	g := &Git{
		upstream: upstream,
		mount:    mount,
		dirs:     dirs,
		user:     user,
		branch:   branch,
	}
	return g
}

func (g *Git) run(args ...string) ([]byte, error) {
	ctx := context.TODO()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.cwd
	cmd.Env = []string{"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null"}
	if g.user != "" {
		uid, gid := osutil.User(g.user)
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	}

	log.Debugf("running in %q as %q %v", cmd.Dir, g.user, cmd.Args)

	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		log.Debug(string(out))
	}
	metricGitOps.Inc()
	if err != nil {
		metricGitFail.Inc()
	}

	return bytes.TrimSpace(out), err
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

	if err := os.MkdirAll(g.mount, 0775); err != nil {
		log.Errorf("Directory %q can not be created", g.mount)
		return fmt.Errorf("failed to create directory %q: %s", g.mount, err)
	}

	uid, gid := osutil.User(g.user)
	if err := os.Chown(g.mount, int(uid), int(gid)); err != nil {
		log.Errorf("Directory %q can not be chown to %q: %s", g.mount, g.user, err)
		return fmt.Errorf("failed to chown directory %q to %q: %s", g.mount, g.user, err)
	}

	g.cwd = ""
	_, err := g.run("clone", "-b", g.branch, "--filter=blob:none", "--no-checkout", "--sparse", g.upstream, g.mount)
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

	out, err := g.run("pull", "--stat", "origin", g.branch)
	if err != nil {
		return false, err
	}
	return g.OfInterest(out), nil
}

// Hash returns the git hash of HEAD in the repo in g.mount. Empty string is returned in case of an error.
// The hash is always truncated to 8 hex digits.
func (g *Git) Hash() string {
	g.cwd = g.mount
	defer func() { g.cwd = "" }()

	out, err := g.run("rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	if len(out) < 8 {
		return ""
	}
	return string(out)[:8]
}

// Rollback checks out commit <hash>, and return nil if no errors are encountered.
func (g *Git) Rollback(hash string) error {
	g.cwd = g.mount
	defer func() { g.cwd = "" }()
	_, err := g.run("checkout", hash)
	return err
}

func (g *Git) Repo() string { return g.mount }
