package converge

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
	"github.com/mlorentedev/dotfiles/cli/internal/update"
)

// DefaultCloneURL is what the checkout step clones when the checkout is
// absent and the caller names no other upstream.
const DefaultCloneURL = "https://github.com/mlorentedev/dotfiles.git"

// checkout makes the dotfiles checkout exist and keeps it level with its
// upstream. Every later step reads the checkout, so it runs first. Absent, it
// is cloned; present, it is fast-forwarded through update.Sync, the same rule
// `dotf update` applies (ADR-019 decision 2): a dirty, diverged, offline or
// upstream-less checkout is left as it is and reported skipped, never merged,
// rebased or reset.
type checkout struct {
	run gitconfig.Runner // gitconfig.ExecRunner in production
	url string
	has func(name string) bool
}

func (checkout) Name() string        { return "checkout" }
func (checkout) Platforms() []string { return nil }

func (c checkout) Reconcile(e Env, dryRun bool) (Result, error) {
	if c.run == nil {
		return Result{Skip: "no git runner is wired into this registry"}, nil
	}
	absent, err := checkoutAbsent(e.RepoRoot)
	if err != nil {
		return Result{}, err
	}
	if absent {
		return c.clone(e, dryRun)
	}
	if err := isDotfilesCheckout(e.RepoRoot); err != nil {
		return Result{}, err
	}
	if !c.has("git") {
		return Result{Skip: "git is not on PATH, so " + e.RepoRoot + " was not fast-forwarded"}, nil
	}
	git := func(args ...string) (string, error) {
		out, err := c.run("git", append([]string{"-C", e.RepoRoot}, args...)...)
		return strings.TrimSpace(string(out)), withStderr(err)
	}
	if dryRun {
		return syncResult(update.Assess(e.RepoRoot, git))
	}
	return syncResult(update.Sync(e.RepoRoot, git))
}

func (c checkout) clone(e Env, dryRun bool) (Result, error) {
	res := Result{
		Changes: 1,
		Detail:  "clone " + c.url + " into " + e.RepoRoot,
		Gate:    "waits for checkout: not cloned yet",
	}
	if !c.has("git") {
		return res, fmt.Errorf("git is not on PATH, so %s cannot be cloned%s", e.RepoRoot, gitHint(e.GOOS))
	}
	if dryRun {
		return res, nil
	}
	if _, err := c.run("git", "clone", "--quiet", c.url, e.RepoRoot); err != nil {
		return res, fmt.Errorf("git clone %s: %w%s", c.url, withStderr(err), gitHint(e.GOOS))
	}
	res.Detail = "cloned " + c.url + " into " + e.RepoRoot
	return res, nil
}

// syncResult reads a Sync or Assess outcome as a step result. Only an
// unexpected fast-forward failure is an error; every state Sync refuses to
// move is the user's to resolve, so it is reported, not failed.
func syncResult(out update.Outcome) (Result, error) {
	switch out.Status {
	case update.StatusCurrent, update.StatusAhead:
		return Result{Detail: out.Message}, nil
	case update.StatusBehind:
		// Only a plan stops here (an apply fast-forwards), and the later steps
		// would read the tree as it is before the fast-forward.
		return Result{
			Changes: 1,
			Detail:  "to apply: fast-forward to " + out.Upstream,
			Gate:    "waits for checkout: fast-forward to " + out.Upstream + " first",
		}, nil
	case update.StatusFastForwarded:
		return Result{Changes: 1, Detail: out.Message}, nil
	case "ff-failed":
		return Result{}, errors.New(out.Message)
	default:
		return Result{Skip: out.Message + "; not fast-forwarded"}, nil
	}
}

// Probe holds the post-condition every later step relies on: the checkout
// exists and is the dotfiles repository.
func (c checkout) Probe(e Env) error {
	if c.run == nil {
		return nil
	}
	return isDotfilesCheckout(e.RepoRoot)
}

// checkoutAbsent reports a path that does not exist, or an empty directory,
// which git clones into as well.
func checkoutAbsent(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return true, nil
	case err != nil:
		return false, fmt.Errorf("reading the checkout %s: %w", path, err)
	}
	return len(entries) == 0, nil
}

// isDotfilesCheckout refuses someone else's repository before git touches it.
func isDotfilesCheckout(path string) error {
	if !env.IsDotfilesCheckout(path) {
		return fmt.Errorf("%s is not a dotfiles checkout (no %s), so it was left untouched; pass --repo or set DOTFILES_REPO_DIR", path, env.CheckoutMarker)
	}
	return nil
}

// withStderr adds git's own complaint to an exit error, which otherwise says
// only "exit status 128".
func withStderr(err error) error {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if msg := strings.TrimSpace(string(ee.Stderr)); msg != "" {
			return fmt.Errorf("%w: %s", err, lastLineOf(msg))
		}
	}
	return err
}

func lastLineOf(s string) string {
	lines := strings.Split(s, "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// gitHint names how to get git where a fresh machine lacks it. On macOS the
// /usr/bin/git shim exists before the Command Line Tools do, and fails.
func gitHint(goos string) string {
	if goos == "darwin" {
		return " (on macOS, install the Command Line Tools: xcode-select --install)"
	}
	return ""
}
