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
}

// isHostProcessInside checks if any running host process has its cwd inside targetPath (Gate f).
func isHostProcessInside(targetPath string) bool {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		isNum := true
		for _, r := range name {
			if r < '0' || r > '9' {
				isNum = false
				break
			}
		}
		if !isNum {
			continue
		}

		dest, err := os.Readlink(filepath.Join("/proc", name, "cwd"))
		if err != nil {
			continue
		}
		absDest, err := filepath.Abs(dest)
		if err != nil {
			continue
		}
		if absDest == absTarget || strings.HasPrefix(absDest, absTarget+string(filepath.Separator)) {
			return true
		}
	}
	return false
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
		DryRun: opts.DryRun,
	}

	cwd, _ := os.Getwd()
	absCwd, _ := filepath.Abs(cwd)

	for _, info := range infos {
		if info.State == StateReapable {
			// Gate (f): never reap current working directory or any worktree with active host processes inside
			absWT, _ := filepath.Abs(info.Path)
			if absWT == absCwd || isHostProcessInside(absWT) {
				report.SkippedCount++
				continue
			}

			if opts.DryRun {
				report.Reaped = append(report.Reaped, info)
				continue
			}

			// TOCTOU check under lock: verify clean status right before removal (fail closed on error)
			dirty, err := runner.IsDirty(info.Path)
			if err != nil || dirty {
				report.SkippedCount++
				continue
			}

			if err := runner.WorktreeRemove(opts.RepoRoot, info.Path); err != nil {
				report.SkippedCount++
				continue
			}

			if info.Branch != "" && !info.IsMain {
				_, _ = runner.BranchDelete(opts.RepoRoot, info.Branch)
			}

			_ = runner.WorktreePrune(opts.RepoRoot)
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
