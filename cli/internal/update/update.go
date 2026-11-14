// Package update implements `dotf update`: the opt-in, scheduler-invoked
// self-deploy that fast-forwards the dotfiles repo and converges the machine
// from it (`dotf converge`, which runs the setup script as its last step on
// Linux and Windows). It is the Go port of scripts/dotfiles-selfupdate.{sh,ps1}
// (CLI-027 / AUDIT-007), collapsing the bash + PowerShell twins into one
// tested path.
//
// The load-bearing contract: every non-actionable condition (not a repo, dirty
// worktree, offline, no upstream, ahead, diverged) is a benign SKIP (nil
// error). Already current runs the system-upgrade step (below), not a skip. The only errors are a real converge failure after a successful
// fast-forward, a failed install of the pinned dotf, and a failed system-upgrade
// step on a day the checkout did not move — so a systemd timer / Scheduled
// Task run reports failure only when something genuinely broke, never for the
// routine "nothing to do".
//
// A day the checkout did not move still runs the system-upgrade converge step
// (TOOL-036), which keeps the catalog's unpinned system packages current. A
// fast-forward converges in full and a reinstall converges with the new
// binary, so both already include it, and a skip runs nothing.
//
// The checkout's DOTF_VERSION pin is part of "up to date" (BUG-144): a
// converge run by a dotf older than the pin lacks the steps the checkout
// declares, and `current` must not be answered while doctor would FAIL the
// drift. The pin is the upstream's (CLI-133): when the running dotf differs
// from it, the pinned release is installed in Go over the running binary on any
// checkout that can be fetched, and a deployable checkout then converges with
// the new binary. install.sh and install.ps1 only bootstrap a machine with no
// dotf yet.
package update

import (
	"fmt"
	"regexp"
	"strings"
)

// Deps abstracts the external surfaces so Run is unit-testable with no real git,
// download or converge. Git runs `git -C <repo> <args...>` and returns trimmed
// stdout (a non-nil error means the git command itself failed). Converge applies
// `dotf converge` to the checkout in this process. Pin reads DOTF_VERSION from
// versions.conf at a git ref ("@{u}", the upstream) or, for "", from the
// checkout's working tree; "" when unpinned. Install puts that release of dotf
// where the running one is and returns its path; ConvergeWith runs `converge`
// with that binary as a child process. Upgrade runs only the system-upgrade
// converge step and returns the step's skip reason, "" when it ran; nil runs
// nothing.
type Deps struct {
	Git          func(args ...string) (string, error)
	Converge     func() error
	Pin          func(ref string) (string, error)
	Install      func(pin string) (exe string, err error)
	ConvergeWith func(exe string) error
	Upgrade      func() (skipped string, err error)
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
	Status   string // stable tag: not-a-repo|dirty|offline|no-upstream|current|ahead|diverged|ff-failed|updated|reinstalled|converge-failed|install-failed|upgrade-failed|upgrade-skipped
	Message  string // human-readable line
	Upstream string // the upstream branch, once resolved (e.g. origin/main)
	Detail   string // multi-line evidence for the status (the dirtying paths), printed after Message
}

// Run executes the self-update against cfg.Repo using the injected Deps. The
// pinned dotf is installed from any checkout that can be fetched, deployable or
// not (CLI-133); only a deployable one (current or fast-forwarded) converges. It
// returns a non-nil error ONLY when the pin cannot be installed, the converge
// fails, or the system-upgrade step fails on a day the checkout is current;
// every other branch is a benign skip (nil error) so a scheduled run does not
// report spurious failures.
func Run(cfg Config, d Deps) (Outcome, error) {
	out := Sync(cfg.Repo, d.Git)
	msg, deployable, fetched := classify(out)
	if !fetched {
		return skip(out.Status, msg)
	}
	pin, note := pinToInstall(cfg.Version, func() (string, string, error) { return pinFor(d, out.Status) })
	if pin != "" {
		return installPin(cfg, d, out.Status, msg, pin, note, deployable)
	}
	if !deployable {
		return skip(out.Status, msg+note)
	}
	if out.Status == StatusCurrent {
		return upgradeQuietDay(d, msg, note)
	}
	// Clean fast-forward landed → converge from it.
	if err := d.Converge(); err != nil {
		return Outcome{Status: "converge-failed", Message: "converge failed — see the report above"},
			fmt.Errorf("converge: %w", err)
	}
	return Outcome{Status: "updated", Message: "self-update complete (fast-forwarded to " + out.Upstream + ")" + note}, nil
}

// classify is the run log line for Sync's status, whether the checkout can be
// deployed (it is level with its upstream), and whether the run got as far as
// the remote: an offline machine or a missing checkout downloads nothing.
func classify(out Outcome) (msg string, deployable, fetched bool) {
	switch out.Status {
	case StatusFastForwarded:
		return out.Message, true, true
	case StatusCurrent:
		return out.Message + " — no deploy needed", true, true
	case StatusAhead:
		// Local commits are not on the upstream, so they are not a deploy: the
		// self-update only ever runs what the upstream holds. It is not a
		// divergence either, so it keeps its own tag (#2205).
		return out.Message + " — nothing to deploy", false, true
	case "not-a-repo":
		return out.Message + " — nothing to self-update", false, false
	case "offline":
		return out.Message + " — skipping self-update", false, false
	case "dirty":
		if out.Detail == "" {
			return out.Message + " — skipping the deploy", false, true
		}
		return out.Message + " — skipping the deploy (commit or stash first). Dirtying paths:\n" + out.Detail, false, true
	}
	return out.Message + " — skipping the deploy", false, true
}

// pinFor reads the pin to install: the upstream's after the fetch, because
// that is what the machine is being updated to, else the checkout's, with a
// note saying so. Assess stops before fetching a dirty checkout, so this
// fetches it; a fetch that fails there is an offline machine, which installs
// nothing ("" pin).
func pinFor(d Deps, status string) (pin, note string, err error) {
	if status == "dirty" {
		if _, ferr := d.Git("fetch", "--quiet"); ferr != nil {
			return "", "\ndotf pin check skipped: git fetch failed (network?)", nil
		}
	}
	if status != "no-upstream" {
		if pin, err = d.Pin("@{u}"); err == nil {
			return pin, "", nil
		}
	}
	pin, err = d.Pin("")
	return pin, "\ndotf pin read from the checkout: the upstream's versions.conf could not be read", err
}

// installPin installs the pinned dotf and, on a deployable checkout, converges
// with it as a child process: converging with this binary is exactly what the
// pin exists to avoid. A checkout that cannot be deployed keeps its status, so
// doctor still reports it stuck, and says the converge waited.
func installPin(cfg Config, d Deps, status, msg, pin, note string, deployable bool) (Outcome, error) {
	exe, err := d.Install(pin)
	if err != nil {
		return Outcome{Status: "install-failed", Message: fmt.Sprintf("dotf %s is not the pinned %s and installing it failed — see the report above", cfg.Version, pin) + note},
			fmt.Errorf("install dotf %s (running %s): %w", pin, cfg.Version, err)
	}
	moved := fmt.Sprintf("dotf %s -> %s (the DOTF_VERSION pin) installed", cfg.Version, pin)
	if !deployable {
		return skip(status, msg+"\n"+moved+"; converge skipped until the checkout can fast-forward"+note)
	}
	if err := d.ConvergeWith(exe); err != nil {
		return Outcome{Status: "converge-failed", Message: moved + "; its converge failed — see the report above" + note},
			fmt.Errorf("converge by the installed dotf %s (running %s): %w", pin, cfg.Version, err)
	}
	return Outcome{Status: "reinstalled", Message: moved + " and converged by the new binary" + note}, nil
}

// upgradeQuietDay runs the system-upgrade step on a day the checkout did not
// move. Its failure is the same class as a converge failure after a
// fast-forward, where the same step would fail the run, so it fails this one
// too, under its own status so doctor can tell it from a broken deploy.
func upgradeQuietDay(d Deps, msg, note string) (Outcome, error) {
	if d.Upgrade == nil {
		return skip(StatusCurrent, msg+note)
	}
	skipped, err := d.Upgrade()
	if err != nil {
		// err already names the step (converge: system-upgrade: ... or
		// converge: cache-gc: ...).
		return Outcome{Status: StatusUpgradeFailed, Message: msg + "; the system-upgrade or cache-gc step failed — see the report above" + note}, err
	}
	if skipped != "" {
		// A skip exits 0 inside converge too: the record is the one place a
		// machine that never upgrades (apt without sudo -n) can show it.
		return skip(StatusUpgradeSkipped, msg+"; the system-upgrade step skipped: "+skipped+note)
	}
	return Outcome{Status: StatusCurrent, Message: msg + "; system-upgrade step converged" + note}, nil
}

// pinToInstall returns the version to install, or "" when none is due, with a
// line for the run log about where the pin came from or why it was not
// checked. A source build never installs (the pin exists to catch a stale
// release, as doctor holds); an unreadable or absent pin is
// not drift. A pin below the running release installs: the pin is the contract.
func pinToInstall(running string, readPin func() (pin, note string, err error)) (string, string) {
	if running == "" {
		return "", ""
	}
	if !releaseVersion.MatchString(running) {
		return "", "\ndotf pin check skipped: the running dotf is a source build (" + running + ")"
	}
	pin, note, err := readPin()
	if err != nil {
		return "", "\ndotf pin check skipped: cannot read DOTF_VERSION: " + err.Error()
	}
	if pin == "" || pin == running {
		return "", ""
	}
	return pin, note
}

// Sync statuses that leave the checkout level with, or ahead of, its upstream.
// Every other status Sync returns names why it left the checkout untouched.
const (
	StatusCurrent       = "current"
	StatusAhead         = "ahead"
	StatusFastForwarded = "fast-forwarded"
	// StatusUpgradeFailed is Run's alone: the checkout was current and the
	// system-upgrade step failed.
	StatusUpgradeFailed = "upgrade-failed"
	// StatusUpgradeSkipped is Run's alone: the checkout was current and the
	// system-upgrade step skipped (a failed refresh, apt without sudo -n).
	StatusUpgradeSkipped = "upgrade-skipped"
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
