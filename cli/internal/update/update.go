// Package update implements `dotf update`: the opt-in, scheduler-invoked
// self-deploy that fast-forwards the dotfiles repo and converges the machine
// from it (`dotf converge`, which runs the setup script as its last step on
// Linux and Windows). It is the Go port of scripts/dotfiles-selfupdate.{sh,ps1}
// (CLI-027 / AUDIT-007), collapsing the bash + PowerShell twins into one
// tested path.
//
// The load-bearing contract: every non-actionable condition (not a repo, dirty
// worktree, offline, no upstream, already current, ahead, diverged) is a benign SKIP
// (nil error). The only errors are a real converge failure after a successful
// fast-forward and a failed install of the pinned dotf — so a systemd timer /
// Scheduled Task run reports failure only when the deploy genuinely broke,
// never for the routine "nothing to do".
//
// The checkout's DOTF_VERSION pin is part of "up to date" (BUG-144): a
// converge run by a dotf older than the pin lacks the steps the checkout
// declares, and `current` must not be answered while doctor would FAIL the
// drift. When the running dotf differs from the pin, the checkout's installer
// installs the pin and converges with the new binary.
package update

import (
	"fmt"
	"regexp"
	"strings"
)

// Deps abstracts the external surfaces so Run is unit-testable with no real git
// or converge. Git runs `git -C <repo> <args...>` and returns trimmed stdout
// (a non-nil error means the git command itself failed). Converge applies
// `dotf converge` to the checkout; the command layer wires it. Pin reads the
// checkout's DOTF_VERSION (after the fast-forward, so it is the new one), ""
// when unpinned. Install runs the checkout's installer for that version, which
// replaces the dotf binary and converges with the new one.
type Deps struct {
	Git      func(args ...string) (string, error)
	Converge func() error
	Pin      func() (string, error)
	Install  func(pin string) error
}

// Config is the resolved run configuration. The converge run is captured by the
// production Converge closure, so it is not a field here.
type Config struct {
	Repo    string // dotfiles checkout to fast-forward (DOTFILES_REPO_DIR)
	Version string // the running dotf's version, as `dotf version` prints it
}

// releaseVersion matches a released dotf version. Anything else ("dev", a
// pseudo-version) is a source build, which the pin does not apply to.
var releaseVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// Outcome classifies a run for the caller to log. Every Outcome except the
// setup-failure path is returned with a nil error.
type Outcome struct {
	Status   string // stable tag: not-a-repo|dirty|offline|no-upstream|current|ahead|diverged|ff-failed|updated|reinstalled|converge-failed|install-failed
	Message  string // human-readable line
	Upstream string // the upstream branch, once resolved (e.g. origin/main)
	Detail   string // multi-line evidence for the status (the dirtying paths), printed after Message
}

// Run executes the self-update against cfg.Repo using the injected Deps. It
// returns a non-nil error ONLY when the converge fails after a clean
// fast-forward or the pinned dotf cannot be installed; every other branch is a
// benign skip (nil error) so a scheduled run does not report spurious failures.
func Run(cfg Config, d Deps) (Outcome, error) {
	out := Sync(cfg.Repo, d.Git)
	switch out.Status {
	case StatusFastForwarded, StatusCurrent:
	case StatusAhead:
		// Local commits are not on the upstream, so they are not a deploy: the
		// self-update only ever runs what the upstream holds. The skip is right,
		// but it is not a divergence: the merge base is the upstream and nothing
		// needs reconciling, so it gets its own tag (#2205).
		return skipDrift(cfg, d, out.Status, out.Message+" — nothing to deploy")
	case "not-a-repo":
		return skip(out.Status, out.Message+" — nothing to self-update")
	case "dirty":
		if out.Detail == "" {
			return skipDrift(cfg, d, out.Status, out.Message+" — skipping")
		}
		return skipDrift(cfg, d, out.Status, out.Message+" — skipping (commit or stash first). Dirtying paths:\n"+out.Detail)
	default:
		return skipDrift(cfg, d, out.Status, out.Message+" — skipping self-update")
	}
	// The checkout is level with its upstream. A running dotf that is not the
	// one it pins would converge short of what the checkout declares (and
	// `current` would hide it), so the pinned one is installed first.
	pin, note := pinToInstall(cfg.Version, d.Pin)
	if pin != "" {
		if err := d.Install(pin); err != nil {
			return Outcome{Status: "install-failed", Message: fmt.Sprintf("dotf %s is not the pinned %s and installing it failed — see the report above", cfg.Version, pin)},
				fmt.Errorf("install dotf %s (running %s): %w", pin, cfg.Version, err)
		}
		// The installer converged with the new binary: converging again with
		// this one is exactly what the pin exists to avoid.
		return Outcome{Status: "reinstalled", Message: fmt.Sprintf("dotf %s -> %s (the DOTF_VERSION pin) installed and converged by the new binary", cfg.Version, pin)}, nil
	}
	if out.Status == StatusCurrent {
		return skip(out.Status, out.Message+" — no deploy needed"+note)
	}
	// Clean fast-forward landed → converge from it.
	if err := d.Converge(); err != nil {
		return Outcome{Status: "converge-failed", Message: "converge failed — see the report above"},
			fmt.Errorf("converge: %w", err)
	}
	return Outcome{Status: "updated", Message: "self-update complete (fast-forwarded to " + out.Upstream + ")" + note}, nil
}

// skipDrift is a skip that installs nothing but names a running dotf that is not
// the checkout's pin: doctor FAILs that drift and sends the owner to `dotf
// update`, so an exit-0 skip must not leave it unsaid.
func skipDrift(cfg Config, d Deps, status, msg string) (Outcome, error) {
	if !releaseVersion.MatchString(cfg.Version) {
		return skip(status, msg)
	}
	pin, err := d.Pin()
	if err != nil || pin == "" || pin == cfg.Version {
		return skip(status, msg)
	}
	return skip(status, msg+"\ndotf "+cfg.Version+" is not the pinned "+pin+
		": `dotf update` installs it once the checkout is clean and level with its upstream, or run ./install.sh from the checkout")
}

// pinToInstall returns the version to install, or "" when none is due, with a
// line for the run log when the check could not be made. A source build never
// installs (the pin exists to catch a stale release, as doctor and the
// installers hold); an unreadable or absent pin is not drift.
func pinToInstall(running string, readPin func() (string, error)) (pin, note string) {
	if running == "" {
		return "", ""
	}
	if !releaseVersion.MatchString(running) {
		return "", "\ndotf pin check skipped: the running dotf is a source build (" + running + ")"
	}
	pin, err := readPin()
	if err != nil {
		return "", "\ndotf pin check skipped: cannot read DOTF_VERSION: " + err.Error()
	}
	if pin == "" || pin == running {
		return "", ""
	}
	return pin, ""
}

// Sync statuses that leave the checkout level with, or ahead of, its upstream.
// Every other status Sync returns names why it left the checkout untouched.
const (
	StatusCurrent       = "current"
	StatusAhead         = "ahead"
	StatusFastForwarded = "fast-forwarded"
	// StatusBehind is Assess's answer for a clean checkout its upstream
	// strictly contains: Sync would fast-forward it. Sync itself never
	// returns it.
	StatusBehind = "behind"
)

// Sync brings the checkout git operates on level with its upstream, by a
// fast-forward and only by one, and reports where that left it. It never
// merges, rebases or resets, and it touches nothing unless the worktree is
// clean and the upstream strictly contains HEAD. `dotf update` (this repo) and
// `dotf harness refresh` (the vault, #2162) share it, so the two cannot
// disagree on what "safe to move" means.
//
// Statuses: not-a-repo|dirty|offline|no-upstream|current|ahead|diverged|
// ff-failed|fast-forwarded. A git command that fails is read as the condition
// it would have ruled out (an unreadable status is "dirty"), never as clean.
func Sync(repo string, git func(args ...string) (string, error)) Outcome {
	out := Assess(repo, git)
	if out.Status != StatusBehind {
		return out
	}
	if _, err := git("merge", "--ff-only", "@{u}"); err != nil {
		return Outcome{Status: "ff-failed", Message: "fast-forward to " + out.Upstream + " failed unexpectedly (worktree left untouched)", Upstream: out.Upstream}
	}
	return Outcome{Status: StatusFastForwarded, Message: "fast-forwarded to " + out.Upstream, Upstream: out.Upstream}
}

// Assess is Sync up to the merge: it fetches, so the upstream it compares is
// current, and reports StatusBehind where Sync would fast-forward. The fetch
// moves remote-tracking refs only; HEAD and the worktree are never touched, so
// a plan can call it.
func Assess(repo string, git func(args ...string) (string, error)) Outcome {
	if _, err := git("rev-parse", "--git-dir"); err != nil {
		return Outcome{Status: "not-a-repo", Message: "not a git repo: " + repo}
	}
	// Never touch a dirty worktree (the primary failure mode). An unreadable
	// status is treated as "cannot confirm clean" (fail-safe).
	out, err := git("status", "--porcelain")
	if err != nil {
		return Outcome{Status: "dirty", Message: "cannot read git status in " + repo + " (fail-safe: cannot confirm the worktree is clean)"}
	}
	// Name the dirtying paths in the message. A scheduled self-update that skips
	// on a dirty worktree is otherwise a silent no-op forever (dotfiles#694): the
	// timer stays green while the deploy never runs. Surfacing the exact paths
	// makes that state diagnosable from the run log alone.
	if dirt := strings.TrimRight(out, "\r\n"); strings.TrimSpace(dirt) != "" {
		return Outcome{Status: "dirty", Message: "dirty worktree in " + repo, Detail: dirt}
	}
	// A fetch failure is transient (network): the caller retries next run.
	if _, err := git("fetch", "--quiet"); err != nil {
		return Outcome{Status: "offline", Message: "git fetch failed in " + repo + " (network?)"}
	}
	upstream, err := git("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		return Outcome{Status: "no-upstream", Message: "no upstream configured for the current branch of " + repo}
	}
	local, err := git("rev-parse", "HEAD")
	if err != nil {
		return Outcome{Status: "no-upstream", Message: "cannot resolve HEAD in " + repo, Upstream: upstream}
	}
	remote, err := git("rev-parse", "@{u}")
	if err != nil {
		return Outcome{Status: "no-upstream", Message: "cannot resolve " + upstream, Upstream: upstream}
	}
	base, err := git("merge-base", "HEAD", "@{u}")
	if err != nil {
		return Outcome{Status: "no-upstream", Message: "cannot compute the merge-base with " + upstream, Upstream: upstream}
	}
	switch {
	case local == remote:
		return Outcome{Status: StatusCurrent, Message: "already current (" + upstream + ")", Upstream: upstream}
	case base == remote:
		return Outcome{Status: StatusAhead, Message: "ahead of " + upstream + " (local commits not pushed yet)", Upstream: upstream}
	case base != local:
		return Outcome{Status: "diverged", Message: "local branch has diverged from " + upstream + " (non fast-forward)", Upstream: upstream}
	}
	return Outcome{Status: StatusBehind, Message: "behind " + upstream, Upstream: upstream}
}

// skip is a tiny helper so every benign branch reads as one line and always
// returns a nil error (the skip contract).
func skip(status, msg string) (Outcome, error) {
	return Outcome{Status: status, Message: msg}, nil
}
