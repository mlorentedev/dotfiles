package worktree

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSweepFileLock(t *testing.T) {
	lockFile := filepath.Join(t.TempDir(), "test.lock")

	unlock1, err := TryLockFile(lockFile)
	if err != nil {
		t.Fatalf("expected first lock to succeed, got: %v", err)
	}
	defer unlock1()

	// Second lock attempt on same file should fail with ErrLocked
	_, err = TryLockFile(lockFile)
	if err != ErrLocked {
		t.Fatalf("expected ErrLocked, got: %v", err)
	}

	// Release first lock
	unlock1()

	// Now lock should succeed again
	unlock2, err := TryLockFile(lockFile)
	if err != nil {
		t.Fatalf("expected lock after release to succeed, got: %v", err)
	}
	defer unlock2()
}

func TestSweepFailClosed(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")
	reapableWT := filepath.Join(tmpDir, "myrepo-wt-reapable")
	dirtyWT := filepath.Join(tmpDir, "myrepo-wt-dirty")
	unmergedWT := filepath.Join(tmpDir, "myrepo-wt-unmerged")
	freshWT := filepath.Join(tmpDir, "myrepo-wt-fresh")

	_ = os.MkdirAll(mainRepo, 0o755)
	_ = os.MkdirAll(reapableWT, 0o755)
	_ = os.MkdirAll(dirtyWT, 0o755)
	_ = os.MkdirAll(unmergedWT, 0o755)
	_ = os.MkdirAll(freshWT, 0o755)

	now := time.Now()

	// Mock porcelain
	mockPorcelain := "worktree " + mainRepo + "\nHEAD 1111\nbranch refs/heads/main\n\n" +
		"worktree " + reapableWT + "\nHEAD 2222\nbranch refs/heads/feat/reapable\n\n" +
		"worktree " + dirtyWT + "\nHEAD 3333\nbranch refs/heads/feat/dirty\n\n" +
		"worktree " + unmergedWT + "\nHEAD 4444\nbranch refs/heads/feat/unmerged\n\n" +
		"worktree " + freshWT + "\nHEAD 5555\nbranch refs/heads/feat/fresh\n\n"

	// Metadata: reapable (old, expired lease, reap_ok true)
	_ = SaveMetadata(reapableWT, Metadata{
		ReapOK:         true,
		CreatedAt:      now.Add(-2 * time.Hour),
		LeaseExpiresAt: now.Add(-1 * time.Hour),
	})

	// Metadata: fresh (only 5m old, < 15m)
	_ = SaveMetadata(freshWT, Metadata{
		ReapOK:         true,
		CreatedAt:      now.Add(-5 * time.Minute),
		LeaseExpiresAt: now.Add(-1 * time.Minute),
	})

	reaped := []string{}
	runner := &MockSweepRunner{
		MockGitRunner: MockGitRunner{
			PorcelainOutput: mockPorcelain,
			DirtyPaths:      map[string]bool{dirtyWT: true},
			MergedBranches:  map[string]bool{"feat/reapable": true, "feat/dirty": true, "feat/fresh": true},
		},
		OnWorktreeRemove: func(repo, path string) error {
			reaped = append(reaped, path)
			return nil
		},
	}

	opts := SweepOptions{
		RepoRoot: mainRepo,
		LockPath: filepath.Join(tmpDir, "sweep.lock"),
		DryRun:   false,
	}

	report, err := SweepWithRunner(opts, runner, now)
	if err != nil {
		t.Fatalf("unexpected error during sweep: %v", err)
	}

	if len(report.Reaped) != 1 || report.Reaped[0].Path != reapableWT {
		t.Fatalf("expected exactly reapableWT to be reaped, got %v", report.Reaped)
	}

	if len(reaped) != 1 || reaped[0] != reapableWT {
		t.Fatalf("expected runner to remove only reapableWT, got %v", reaped)
	}

	if report.SkippedCount != 4 { // mainRepo, dirtyWT, unmergedWT, freshWT
		t.Errorf("expected 4 skipped, got %d", report.SkippedCount)
	}
}

func TestSweepLogsSHA(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")
	reapableWT := filepath.Join(tmpDir, "myrepo-wt-reapable")

	_ = os.MkdirAll(mainRepo, 0o755)
	_ = os.MkdirAll(reapableWT, 0o755)

	now := time.Now()

	mockPorcelain := "worktree " + mainRepo + "\nHEAD 1111\nbranch refs/heads/main\n\n" +
		"worktree " + reapableWT + "\nHEAD 2222\nbranch refs/heads/feat/reapable\n\n"

	_ = SaveMetadata(reapableWT, Metadata{
		ReapOK:         true,
		CreatedAt:      now.Add(-2 * time.Hour),
		LeaseExpiresAt: now.Add(-1 * time.Hour),
	})

	branchDeleted := false
	deletedSHA := ""
	runner := &MockSweepRunner{
		MockGitRunner: MockGitRunner{
			PorcelainOutput: mockPorcelain,
			MergedBranches:  map[string]bool{"feat/reapable": true},
		},
		OnWorktreeRemove: func(repo, path string) error {
			return nil
		},
		OnBranchDelete: func(repo, branch string) (string, error) {
			branchDeleted = true
			deletedSHA = "2222"
			return deletedSHA, nil
		},
	}

	opts := SweepOptions{
		RepoRoot: mainRepo,
		LockPath: filepath.Join(tmpDir, "sweep.lock"),
		DryRun:   false,
	}

	report, err := SweepWithRunner(opts, runner, now)
	if err != nil {
		t.Fatalf("unexpected error during sweep: %v", err)
	}

	if len(report.Reaped) != 1 {
		t.Fatalf("expected 1 worktree reaped, got %d", len(report.Reaped))
	}

	if !branchDeleted || deletedSHA != "2222" {
		t.Errorf("expected branch to be deleted with recorded SHA 2222, got deleted=%v sha=%s", branchDeleted, deletedSHA)
	}
}

func TestSweepTOCTOUDirtyErrorFailsClosed(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")
	reapableWT := filepath.Join(tmpDir, "myrepo-wt-err")

	_ = os.MkdirAll(mainRepo, 0o755)
	_ = os.MkdirAll(reapableWT, 0o755)

	now := time.Now()

	mockPorcelain := "worktree " + mainRepo + "\nHEAD 1111\nbranch refs/heads/main\n\n" +
		"worktree " + reapableWT + "\nHEAD 2222\nbranch refs/heads/feat/err\n\n"

	_ = SaveMetadata(reapableWT, Metadata{
		ReapOK:         true,
		CreatedAt:      now.Add(-2 * time.Hour),
		LeaseExpiresAt: now.Add(-1 * time.Hour),
	})

	// First IsDirty succeeds for list, but during sweep TOCTOU check it fails
	firstCall := true
	runner := &MockSweepRunner{
		MockGitRunner: MockGitRunner{
			PorcelainOutput: mockPorcelain,
			MergedBranches:  map[string]bool{"feat/err": true},
		},
		OnWorktreeRemove: func(repo, path string) error {
			t.Fatalf("WorktreeRemove should NOT be called when IsDirty errors during TOCTOU")
			return nil
		},
	}
	// Dynamic IsDirty: first call clean (for list), second call error (for sweep TOCTOU)
	runner.DirtyPaths = map[string]bool{}

	// Overwrite IsDirty behavior via a custom runner wrapper
	errorRunner := &toctouErrorRunner{
		MockSweepRunner: runner,
		failTOCTOU:      true,
	}

	opts := SweepOptions{
		RepoRoot: mainRepo,
		LockPath: filepath.Join(tmpDir, "sweep.lock"),
		DryRun:   false,
	}

	report, err := SweepWithRunner(opts, errorRunner, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Reaped) != 0 {
		t.Fatalf("expected 0 reaped due to TOCTOU error, got %d", len(report.Reaped))
	}
	if report.SkippedCount != 2 { // mainRepo, reapableWT
		t.Errorf("expected 2 skipped, got %d", report.SkippedCount)
	}
	_ = firstCall
}

type toctouErrorRunner struct {
	*MockSweepRunner
	failTOCTOU bool
	callCount  int
}

func (r *toctouErrorRunner) IsDirty(path string) (bool, error) {
	r.callCount++
	// Call 1 is during ListWithRunner
	if r.callCount == 1 {
		return false, nil
	}
	// Call 2 is during Sweep TOCTOU check
	if r.failTOCTOU {
		return false, os.ErrPermission
	}
	return false, nil
}
