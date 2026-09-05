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
}

// hostProcessInside is Gate f. Its implementation lives in
// sweep_proc_linux.go / sweep_proc_other.go, because it is the one gate with no
// portable form -- see those files for why the unsupported answer is `true`.
//
// Indirected through a variable so a test can drive both answers on any
// platform. Without the seam the refusing branch is reachable only on a machine
// the CI does not have, which is how the defect this replaced survived review.
var hostProcessInside = isHostProcessInside

func isCandidateForReap(info Info, absCwd string) bool {
	if info.State != StateReapable {
		return false
	}
	// Gate (f): never reap current working directory or any worktree with active host processes inside
	absWT, err := filepath.Abs(info.Path)
	if err != nil || absWT == absCwd || hostProcessInside(absWT) {
		return false
	}
	return true
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
	// TOCTOU checks under lock: host process/cwd guard, clean status, merge status, lease expiration
	absWT, err := filepath.Abs(info.Path)
	if err != nil || absWT == absCwd || hostProcessInside(absWT) {
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
		if !isCandidateForReap(info, absCwd) {
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
