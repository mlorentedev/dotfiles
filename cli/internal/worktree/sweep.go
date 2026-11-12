package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SweepRunner interface {
	GitRunner
	WorktreeRemove(repoRoot, worktreePath string) error
	// WorktreeRemoveForce removes a worktree whose untracked content is known
	// to be disposable: only a review worktree past its lease.
	WorktreeRemoveForce(repoRoot, worktreePath string) error
	BranchDelete(repoRoot, branch string) (string, error)
	WorktreePrune(repoRoot string) error
}

type RealSweepRunner struct {
	RealGitRunner
}

func (r *RealSweepRunner) WorktreeRemove(repoRoot, worktreePath string) error {
	cmd := gitIn(repoRoot, "worktree", "remove", worktreePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove failed (%w): %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *RealSweepRunner) WorktreeRemoveForce(repoRoot, worktreePath string) error {
	cmd := gitIn(repoRoot, "worktree", "remove", "--force", worktreePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove --force failed (%w): %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (r *RealSweepRunner) BranchDelete(repoRoot, branch string) (string, error) {
	// First resolve exact full SHA for instant recovery logging (AC5)
	shaCmd := gitIn(repoRoot, "rev-parse", branch)
	shaOut, err := shaCmd.Output()
	sha := strings.TrimSpace(string(shaOut))
	if err != nil || sha == "" {
		sha = "unknown"
	}

	// Log before deletion for instant undoability (AC5)
	_, _ = fmt.Fprintf(os.Stderr, "Deleting branch %s (was %s)\n", branch, sha)

	delCmd := gitIn(repoRoot, "branch", "-D", branch)
	if out, err := delCmd.CombinedOutput(); err != nil {
		return sha, fmt.Errorf("git branch -D failed (%w): %s", err, strings.TrimSpace(string(out)))
	}
	return sha, nil
}

func (r *RealSweepRunner) WorktreePrune(repoRoot string) error {
	cmd := gitIn(repoRoot, "worktree", "prune")
	return cmd.Run()
}

type MockSweepRunner struct {
	MockGitRunner
	OnWorktreeRemove      func(repoRoot, worktreePath string) error
	OnWorktreeRemoveForce func(repoRoot, worktreePath string) error
	OnBranchDelete        func(repoRoot, branch string) (string, error)
}

func (m *MockSweepRunner) WorktreeRemoveForce(repoRoot, worktreePath string) error {
	if m.OnWorktreeRemoveForce != nil {
		return m.OnWorktreeRemoveForce(repoRoot, worktreePath)
	}
	return nil
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
	// Sessions are the harness session stores: sweep skips a worktree with a
	// session there under SessionWindow old. It has no override; done
	// --ignore-recent-session is the deliberate path.
	Sessions SessionStores
	// LandedHeads answers which merged pull requests had a branch as their
	// head (#2818). Nil asks GitHub through gh.
	LandedHeads LandedHeads
}

// SessionSkip is a worktree sweep left alone for a harness session, with
// the transcript or the read error that decided it.
type SessionSkip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type SweepReport struct {
	Reaped       []Info `json:"reaped"`
	SkippedCount int    `json:"skipped_count"`
	// SessionSkips names each worktree counted in SkippedCount for a recent
	// or uninspectable session: a false refusal (a sibling slug) is only
	// visible to an operator who sees which worktree it was.
	SessionSkips []SessionSkip `json:"session_skips,omitempty"`
	// MergeUnknownSkips names each worktree kept only because the forge could
	// not say whether its branch landed (#2873).
	MergeUnknownSkips []SessionSkip `json:"merge_unknown_skips,omitempty"`
	// LaneSkips names each named lane counted in SkippedCount that sweep did
	// not park, with the reason: a dirty tree, a landing it could not confirm,
	// or a detach git refused (#3263).
	LaneSkips []SessionSkip `json:"lane_skips,omitempty"`
	DryRun    bool          `json:"dry_run"`
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
	// The cwd check is the same within-or-equal test done uses, on resolved
	// paths: equality alone missed a caller in a subdirectory, or one that
	// reached the worktree through a symlink.
	absWT, err := filepath.Abs(info.Path)
	if err != nil || pathWithin(resolvedPath(absCwd), resolvedPath(absWT)) {
		return GateFReading{}, false
	}
	reading := hostProcessInside(absWT)
	return reading, !reading.Inside
}

func isDirty(path string, runner SweepRunner) bool {
	dirt, err := runner.Dirt(path)
	return err != nil || dirt.Any()
}

// laneHeldByDirt reports whether a named lane is dirty and would otherwise
// have reached the park decision: Classify answers StateDirty before it looks
// at the lease, so a lane in use with a live lease, a lane younger than the
// park age or a parked lane is dirty too, and none of those is a lane sweep
// declined to park (#3372).
func laneHeldByDirt(info Info, now time.Time) bool {
	if info.State != StateDirty || info.Metadata == nil || info.Metadata.Lane == "" {
		return false
	}
	return !strings.EqualFold(info.Metadata.State, string(StateParked)) &&
		!info.Metadata.LeaseExpiresAt.After(now) &&
		now.Sub(info.Metadata.CreatedAt) >= 15*time.Minute
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
	if err != nil || pathWithin(resolvedPath(absCwd), resolvedPath(absWT)) {
		return false
	}
	// Re-read under the lock, like the lease below: the kind decides which
	// gates apply, so it is not taken from the listing.
	if meta, err := LoadMetadata(info.Path); err == nil && meta != nil && meta.Kind == KindReview {
		return reapReviewWorktree(repoRoot, info, meta, absWT, runner, now)
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

// reapReviewWorktree removes a review worktree once its lease has passed
// (HARNESS-142). It skips the merged gate, since the worktree has no branch,
// but not the clean gate's purpose: uncommitted content is removed only when
// `spec deadline` recorded that it copied the review's output back
// (OutputSaved), so what --force discards is what else the reviewer left. A
// review whose output was never copied back is kept, and so is a tree whose
// state cannot be read. Gate f stays last: a reviewer still running past its
// lease is the one case the lease cannot see.
func reapReviewWorktree(repoRoot string, info Info, meta *Metadata, absWT string, runner SweepRunner, now time.Time) bool {
	if !isLeaseExpired(info.Path, now) {
		return false
	}
	dirt, err := runner.Dirt(info.Path)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: keeping review worktree %s: its state cannot be read: %v\n", info.Path, err)
		return false
	}
	if dirt.Any() && !meta.OutputSaved {
		return false
	}
	if hostProcessInside(absWT).Inside {
		return false
	}
	if dirt.Any() {
		_, _ = fmt.Fprintf(os.Stderr, "Removing review worktree %s: its output was copied back, and what else the reviewer left in it is discarded\n", info.Path)
	}
	if err := runner.WorktreeRemoveForce(repoRoot, info.Path); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: removing review worktree %s: %v\n", info.Path, err)
		return false
	}
	_ = runner.WorktreePrune(repoRoot)
	return true
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
		if info.State == StateParkable {
			if reason := sessionSkipReason(opts.Sessions, info.Path, now); reason != "" {
				report.SkippedCount++
				report.SessionSkips = append(report.SessionSkips, SessionSkip{Path: info.Path, Reason: reason})
				continue
			}
			if opts.DryRun {
				report.Reaped = append(report.Reaped, info)
				continue
			}
			if parked, why := parkSingleNamedLane(opts.RepoRoot, info, runner, opts.LandedHeads); parked {
				report.Reaped = append(report.Reaped, info)
			} else {
				report.SkippedCount++
				report.LaneSkips = append(report.LaneSkips, SessionSkip{Path: info.Path, Reason: why})
			}
			continue
		}
		reading, candidate := gateF(info, absCwd)
		if reading.Uninspectable > report.UninspectableProcesses {
			report.UninspectableProcesses = reading.Uninspectable
		}
		if !candidate {
			report.SkippedCount++
			if laneHeldByDirt(info, now) {
				report.LaneSkips = append(report.LaneSkips, SessionSkip{Path: info.Path, Reason: info.StateReason})
			}
			if info.PRMergedUnknown != "" && (info.State == StateUnmerged || info.State == StateOrphan) {
				report.MergeUnknownSkips = append(report.MergeUnknownSkips, SessionSkip{Path: info.Path, Reason: info.StateReason})
			}
			continue
		}
		// After gate f, so only a worktree sweep would otherwise reap is
		// reported: the main checkout's key prefixes every legacy sibling's.
		if reason := sessionSkipReason(opts.Sessions, info.Path, now); reason != "" {
			report.SkippedCount++
			report.SessionSkips = append(report.SessionSkips, SessionSkip{Path: info.Path, Reason: reason})
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

func parkSingleNamedLane(repoRoot string, info Info, runner SweepRunner, landedHeads LandedHeads) (parked bool, why string) {
	if dirt, err := runner.Dirt(info.Path); err != nil {
		return false, "its state could not be read: " + err.Error()
	} else if dirt.Any() {
		return false, "has uncommitted changes or untracked files"
	}
	if landed, why := laneLanded(repoRoot, info, runner, landedHeads); !landed {
		return false, why
	}
	// Write State="PARKED" before checkout --detach / branch delete (M-CODE-2).
	parkedMeta, issueNum := parkedLease(info.Path, info.Metadata)
	if err := SaveMetadata(info.Path, parkedMeta); err != nil {
		return false, "its lease could not be written: " + err.Error()
	}
	if _, out, err := detachAtBase(repoRoot, info.Path, false, info.Metadata); err != nil {
		return false, "git refused to detach it: " + gitFailure(out, err)
	}
	deleteBranchIfSafe(repoRoot, info.Branch, info.IsMain, runner)
	releaseParkedLane(repoRoot, info.Path, info.Branch, issueNum)
	return true, ""
}

// gitFailure is the first line git said when it refused, or its exit status
// when it said nothing: the report prints one line per lane.
func gitFailure(out []byte, err error) string {
	if msg, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n"); msg != "" {
		return msg
	}
	return err.Error()
}

// laneLanded is the landing decision parkSingleNamedLane acts on, kept apart
// from the parking itself. Ancestry alone cannot see a squash merge, so a
// branch git does not call merged is asked of the forge (the caller's
// LandedHeads, or gh); a landing nobody could confirm is reported as such,
// never resolved to "not landed".
func laneLanded(repoRoot string, info Info, runner SweepRunner, landedHeads LandedHeads) (bool, string) {
	merged := isMerged(repoRoot, info.Branch, runner)
	heads := landedHeads
	if !merged && heads == nil {
		heads = mergedPRHeads
	}
	if heads == nil {
		if merged {
			return true, ""
		}
		// No reader at all: nothing can say the work landed, so it did not.
		return false, "its landing could not be decided"
	}
	_, _, landed, why := branchLanding(repoRoot, info.Path, heads)
	if !landed && why == "" {
		why = "its landing could not be decided"
	}
	return landed, why
}

func Sweep(opts SweepOptions) (*SweepReport, error) {
	return SweepWithRunner(opts, realSweepRunner(opts), time.Now())
}

func realSweepRunner(opts SweepOptions) *RealSweepRunner {
	return &RealSweepRunner{RealGitRunner: RealGitRunner{LandedHeads: opts.LandedHeads}}
}
