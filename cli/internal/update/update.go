// Package update implements `dotf update`: the opt-in, scheduler-invoked
// self-deploy that fast-forwards the dotfiles repo and re-runs the idempotent
// setup. It is the Go port of scripts/dotfiles-selfupdate.{sh,ps1} (CLI-027 /
// AUDIT-007), collapsing the bash + PowerShell twins into one tested path.
//
// The load-bearing contract: every non-actionable condition (not a repo, dirty
// worktree, offline, no upstream, already current, diverged) is a benign SKIP
// (nil error). The ONLY error is a real setup failure after a successful
// fast-forward — so a systemd timer / Scheduled Task run reports failure only
// when the deploy genuinely broke, never for the routine "nothing to do".
package update

import (
	"fmt"
	"strings"
)

// Deps abstracts the external surfaces so Run is unit-testable with no real git
// or setup exec. Git runs `git -C <repo> <args...>` and returns trimmed stdout
// (a non-nil error means the git command itself failed). RunSetup executes the
// setup command. Production wires these to os/exec (OS-aware for the setup shell)
// in the command layer.
type Deps struct {
	Git      func(args ...string) (string, error)
	RunSetup func() error
}

// Config is the resolved run configuration. The setup command is captured by the
// production RunSetup closure, so it is not a field here.
type Config struct {
	Repo string // dotfiles checkout to fast-forward (DOTFILES_REPO_DIR)
}

// Outcome classifies a run for the caller to log. Every Outcome except the
// setup-failure path is returned with a nil error.
type Outcome struct {
	Status  string // stable tag: not-a-repo|dirty|offline|no-upstream|current|diverged|ff-failed|updated|setup-failed
	Message string // human-readable line
}

// Run executes the self-update against cfg.Repo using the injected Deps. It
// returns a non-nil error ONLY when the idempotent setup fails after a clean
// fast-forward; every other branch is a benign skip (nil error) so a scheduled
// run does not report spurious failures.
func Run(cfg Config, d Deps) (Outcome, error) {
	if _, err := d.Git("rev-parse", "--git-dir"); err != nil {
		return skip("not-a-repo", "not a git repo: "+cfg.Repo+" — nothing to self-update")
	}
	// Never touch a dirty worktree (the primary failure mode). An unreadable
	// status is treated as "cannot confirm clean" → skip (fail-safe).
	out, err := d.Git("status", "--porcelain")
	if err != nil {
		return skip("dirty", "cannot read git status in "+cfg.Repo+" — skipping (fail-safe: cannot confirm the worktree is clean)")
	}
	// Name the dirtying paths in the message. A scheduled self-update that skips
	// on a dirty worktree is otherwise a silent no-op forever (dotfiles#694): the
	// timer stays green while the deploy never runs. Surfacing the exact paths
	// makes that state diagnosable from the run log alone.
	if dirt := strings.TrimSpace(out); dirt != "" {
		return skip("dirty", "dirty worktree in "+cfg.Repo+" — skipping (commit or stash first). Dirtying paths:\n"+dirt)
	}
	// A fetch failure is transient (network) → skip and retry next slot.
	if _, err := d.Git("fetch", "--quiet"); err != nil {
		return skip("offline", "git fetch failed (network?) — skipping self-update this run")
	}
	upstream, err := d.Git("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		return skip("no-upstream", "no upstream configured for the current branch — skipping")
	}
	local, err := d.Git("rev-parse", "HEAD")
	if err != nil {
		return skip("no-upstream", "cannot resolve HEAD — skipping")
	}
	remote, err := d.Git("rev-parse", "@{u}")
	if err != nil {
		return skip("no-upstream", "cannot resolve "+upstream+" — skipping")
	}
	base, err := d.Git("merge-base", "HEAD", "@{u}")
	if err != nil {
		return skip("no-upstream", "cannot compute merge-base — skipping")
	}
	if local == remote {
		return skip("current", "already current ("+upstream+") — no deploy needed")
	}
	// Diverged / non-fast-forward: never merge, rebase, or reset unattended.
	if base != local {
		return skip("diverged", "local branch has diverged from "+upstream+" (non fast-forward) — skipping")
	}
	if _, err := d.Git("merge", "--ff-only", "@{u}"); err != nil {
		return skip("ff-failed", "fast-forward failed unexpectedly — skipping (worktree left untouched)")
	}
	// Clean fast-forward landed → re-run the idempotent setup. THE only error path.
	if err := d.RunSetup(); err != nil {
		return Outcome{Status: "setup-failed", Message: "setup failed — see output above"},
			fmt.Errorf("setup: %w", err)
	}
	return Outcome{Status: "updated", Message: "self-update complete (fast-forwarded to " + upstream + ")"}, nil
}

// skip is a tiny helper so every benign branch reads as one line and always
// returns a nil error (the skip contract).
func skip(status, msg string) (Outcome, error) {
	return Outcome{Status: status, Message: msg}, nil
}
