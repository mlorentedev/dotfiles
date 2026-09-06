package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type SweepRunner interface {
	GitRunner
	WorktreeRemove(repoRoot, worktreePath string) error
	BranchDelete(repoRoot, branch string) (string, error)
	WorktreePrune(repoRoot string) error
}

type RealSweepRunner struct {
	RealGitRunner
}

func (r *RealSweepRunner) WorktreeRemove(repoRoot, worktreePath string) error {
	cmd := exec.Command("git", "-C", repoRoot, "worktree", "remove", worktreePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove failed (%w): %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *RealSweepRunner) BranchDelete(repoRoot, branch string) (string, error) {
	// First resolve exact full SHA for instant recovery logging (AC5)
	shaCmd := exec.Command("git", "-C", repoRoot, "rev-parse", branch)
	shaOut, err := shaCmd.Output()
	sha := strings.TrimSpace(string(shaOut))
	if err != nil || sha == "" {
		sha = "unknown"
	}

	// Log before deletion for instant undoability (AC5)
	_, _ = fmt.Fprintf(os.Stderr, "Deleting branch %s (was %s)\n", branch, sha)

	delCmd := exec.Command("git", "-C", repoRoot, "branch", "-D", branch)
	if out, err := delCmd.CombinedOutput(); err != nil {
		return sha, fmt.Errorf("git branch -D failed (%w): %s", err, strings.TrimSpace(string(out)))
	}
	return sha, nil
}

func (r *RealSweepRunner) WorktreePrune(repoRoot string) error {
	cmd := exec.Command("git", "-C", repoRoot, "worktree", "prune")
	return cmd.Run()
}

type MockSweepRunner struct {
	MockGitRunner
	OnWorktreeRemove func(repoRoot, worktreePath string) error
	OnBranchDelete   func(repoRoot, branch string) (string, error)
}

func (m *MockSweepRunner) WorktreeRemove(repoRoot, worktreePath string) error {
	if m.OnWorktreeRemove != nil {
		return m.OnWorktreeRemove(repoRoot, worktreePath)
	}
	return nil
}

func (m *MockSweepRunner) BranchDelete(repoRoot, branch string) (string, error) {
	if m.OnBranchDelete != nil {
		return m.OnBranchDelete(repoRoot, branch)
	}
	return "mock-sha", nil
}

func (m *MockSweepRunner) WorktreePrune(repoRoot string) error {
	return nil
}

type SweepOptions struct {
	RepoRoot string
	LockPath string
	DryRun   bool
}

type SweepReport struct {
	Reaped       []Info `json:"reaped"`
	SkippedCount int    `json:"skipped_count"`
	DryRun       bool   `json:"dry_run"`
	// ProcessDiscovery is false when Gate f has no implementation on this
	// platform and is therefore refusing everything. Reported because an inert
	// sweep prints "reaped 0" exactly like a machine with nothing to clean up,
	// and those are different facts.
	ProcessDiscovery bool `json:"process_discovery"`
	// UninspectableProcesses is the largest number of processes any single
	// worktree's scan could not read the cwd of.
	//
	// It states Gate f's REACH; it is not an anomaly signal, and an earlier
	// version of this comment claimed it was. The number is non-zero on every
	// Linux machine by construction: /proc/<pid>/cwd is gated by
	// ptrace_may_access, so a scan run as an ordinary user can never read
	// root's processes, another user's, or same-uid ones that set
	// PR_SET_DUMPABLE=0 (systemd --user, ssh-agent, browser sandboxes). One
	// desktop measurement: 364 of 571 processes out of reach, 20 of them the
	// caller's own. A count that can never be zero cannot warn about anything.
	//
	// Reported rather than acted on: refusing on it would make sweep
	// permanently inert on Linux, which is the same trap as answering "nobody
	// is inside" — it makes the tool useless instead of dangerous, but it makes
	// it useless every single run.
	UninspectableProcesses int `json:"uninspectable_processes"`
}

// GateFReading is what Gate f could actually establish about one worktree.
//
// It is a struct and not a bool because the scan has three outcomes, not two.
// A process whose cwd cannot be read is neither inside nor outside: reading
// /proc/<pid>/cwd is gated by ptrace_may_access, so an ordinary user's scan
// cannot see root's cwd, another user's, or a same-uid process that made itself
// non-dumpable. Collapsing that into "not inside" is the fail-open an
// adversarial review caught here; collapsing it into "inside" would make sweep
// permanently inert, since /proc/1/cwd is unreadable to every non-root caller.
//
// The third option is to keep the fact and be honest about what it is: a
// statement of how far the gate can see, not a warning that something is wrong.
// See the SweepReport field for the measurement.
type GateFReading struct {
	Inside bool
	// Uninspectable is a coverage figure, never a threshold. Nothing decides on
	// it, because nothing can: an unreadable process stays unreadable however
	// many of them there are.
	Uninspectable int
}

// hostProcessInside is Gate f. Its implementation lives in
// sweep_proc_linux.go / sweep_proc_other.go, because it is the one gate with no
// portable form -- see those files for why the unsupported answer is `true`.
//
// Indirected through a variable so a test can drive every answer on any
// platform. Without the seam the refusing branch is reachable only on a machine
// the CI does not have, which is how the defect this replaced survived review.
var hostProcessInside = isHostProcessInside

// gateF returns the reading alongside the decision, so a caller that wants to
// report how far the scan reached does not have to run it twice.
//
// This is the function production calls. It used to have a bool-returning
// wrapper, isCandidateForReap, which no caller outside the tests used — so the
// tests that pinned Gate f's refusal were pinning a function that could not
// affect a deletion. Removed rather than kept for convenience: a test seam into
// dead code reports coverage it does not have.
func gateF(info Info, absCwd string) (GateFReading, bool) {
	if info.State != StateReapable {
		return GateFReading{}, false
	}
	// Gate (f): never reap current working directory or any worktree with active host processes inside
	absWT, err := filepath.Abs(info.Path)
	if err != nil || absWT == absCwd {
		return GateFReading{}, false
	}
	reading := hostProcessInside(absWT)
	return reading, !reading.Inside
}

func isDirty(path string, runner SweepRunner) bool {
	dirty, err := runner.IsDirty(path)
	return err != nil || dirty
}

func isMerged(repoRoot, branch string, runner SweepRunner) bool {
	merged, err := runner.IsPRMerged(repoRoot, branch)
	return err == nil && merged
}

func isLeaseExpired(path string, now time.Time) bool {
	meta, err := LoadMetadata(path)
	if err != nil || meta == nil {
		return false
	}
	return meta.ReapOK && !meta.LeaseExpiresAt.After(now)
}

func deleteBranchIfSafe(repoRoot, branch string, isMain bool, runner SweepRunner) {
	if branch == "" || isMain {
		return
	}
	if _, err := runner.BranchDelete(repoRoot, branch); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: deleting branch %s: %v\n", branch, err)
	}
}

func executeWorktreeReap(repoRoot string, info Info, runner SweepRunner) bool {
	if err := runner.WorktreeRemove(repoRoot, info.Path); err != nil {
		return false
	}
	deleteBranchIfSafe(repoRoot, info.Branch, info.IsMain, runner)
	_ = runner.WorktreePrune(repoRoot)
	return true
}

func reapSingleWorktree(repoRoot string, info Info, runner SweepRunner, now time.Time, absCwd string) bool {
	// TOCTOU checks under lock: clean status, merge status, lease expiration,
	// and LAST the host-process guard.
	//
	// The order is the point, not a style choice. isDirty and isMerged shell out
	// to git, so a Gate f check placed before them leaves that latency inside
	// the window between "nobody is in there" and the removal -- a shell that
	// cds in while git is running would be missed. Gate f goes immediately
	// before executeWorktreeReap so the window is as narrow as this code can
	// make it. TestGateFIsTheLastCheckBeforeRemoval pins the order.
	//
	// It is also the cheaper order: Gate f walks all of /proc, so running it
	// after the cheap refusals means fewer walks, not more.
	absWT, err := filepath.Abs(info.Path)
	if err != nil || absWT == absCwd {
		return false
	}
	if isDirty(info.Path, runner) {
		return false
	}
	if !isMerged(repoRoot, info.Branch, runner) {
		return false
	}
	if !isLeaseExpired(info.Path, now) {
		return false
	}
	// Last, and deliberately so: see the ordering note above.
	if hostProcessInside(absWT).Inside {
		return false
	}
	return executeWorktreeReap(repoRoot, info, runner)
}

func SweepWithRunner(opts SweepOptions, runner SweepRunner, now time.Time) (*SweepReport, error) {
	lockPath := opts.LockPath
	if lockPath == "" {
		lockPath = DefaultLockPath()
	}

	unlock, err := TryLockFile(lockPath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	infos, err := ListWithRunner(opts.RepoRoot, runner, now)
	if err != nil {
		return nil, err
	}

	report := &SweepReport{
		DryRun:           opts.DryRun,
		ProcessDiscovery: processDiscoverySupported,
	}

	cwd, _ := os.Getwd()
	absCwd, _ := filepath.Abs(cwd)

	for _, info := range infos {
		reading, candidate := gateF(info, absCwd)
		if reading.Uninspectable > report.UninspectableProcesses {
			report.UninspectableProcesses = reading.Uninspectable
		}
		if !candidate {
			report.SkippedCount++
			continue
		}

		if opts.DryRun {
			report.Reaped = append(report.Reaped, info)
			continue
		}

		if reapSingleWorktree(opts.RepoRoot, info, runner, now, absCwd) {
			report.Reaped = append(report.Reaped, info)
		} else {
			report.SkippedCount++
		}
	}

	return report, nil
}

func Sweep(opts SweepOptions) (*SweepReport, error) {
	return SweepWithRunner(opts, &RealSweepRunner{}, time.Now())
}
