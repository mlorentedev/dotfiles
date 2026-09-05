package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoneTeardown(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")

	// 1. Refuse teardown if path is main repository
	err := Done(DoneOptions{
		RepoRoot:     mainRepo,
		WorktreePath: mainRepo,
	})
	if err == nil {
		t.Errorf("expected Done on main repo to fail")
	}

	// 2. Error if worktreePath is empty
	err = Done(DoneOptions{
		RepoRoot:     mainRepo,
		WorktreePath: "",
	})
	if err == nil {
		t.Errorf("expected empty worktreePath to fail")
	}
}

func setupTestGitRepoAndWorktree(t *testing.T) (repoDir, wtDir, lockPath string) {
	tmpDir := t.TempDir()
	repoDir = filepath.Join(tmpDir, "repo")
	wtDir = filepath.Join(tmpDir, "repo-wt-feat")
	lockPath = filepath.Join(tmpDir, "test.lock")

	runGit := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, string(out))
		}
	}

	_ = os.MkdirAll(repoDir, 0o755)
	runGit(repoDir, "init", "-b", "main")
	runGit(repoDir, "config", "user.name", "Test User")
	runGit(repoDir, "config", "user.email", "test@example.com")
	_ = os.WriteFile(filepath.Join(repoDir, "initial.txt"), []byte("hello"), 0o644)
	runGit(repoDir, "add", "initial.txt")
	runGit(repoDir, "commit", "-m", "initial commit")

	runGit(repoDir, "worktree", "add", "-b", "feat/test", wtDir)
	return repoDir, wtDir, lockPath
}

func TestDoneRefusesUnpushedCommitsWithNoUpstream(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)

	// Add an unpushed commit on feat/test (which has no upstream)
	filePath := filepath.Join(wtDir, "wip.txt")
	_ = os.WriteFile(filePath, []byte("unpushed commit content"), 0o644)
	cmd := exec.Command("git", "add", "wip.txt")
	cmd.Dir = wtDir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "local commit without upstream")
	cmd.Dir = wtDir
	_ = cmd.Run()

	// 1. Done without force must refuse to remove
	err := Done(DoneOptions{
		RepoRoot:     repoDir,
		WorktreePath: wtDir,
		LockPath:     lockPath,
		Force:        false,
	})
	if err == nil {
		t.Fatalf("expected Done to fail on branch with unpushed commits and no upstream, but got nil")
	}
	if !strings.Contains(err.Error(), "unpushed commit") {
		t.Errorf("expected error to mention unpushed commit, got: %v", err)
	}

	// Verify worktree still exists
	if _, statErr := os.Stat(wtDir); os.IsNotExist(statErr) {
		t.Errorf("worktree directory was deleted despite unpushed commits!")
	}

	// 2. Done with force must succeed
	err = Done(DoneOptions{
		RepoRoot:     repoDir,
		WorktreePath: wtDir,
		LockPath:     lockPath,
		Force:        true,
	})
	if err != nil {
		t.Fatalf("expected Done with Force=true to succeed, got: %v", err)
	}

	// Verify worktree is removed
	if _, statErr := os.Stat(wtDir); !os.IsNotExist(statErr) {
		t.Errorf("expected worktree directory to be removed after force Done")
	}
}

func TestDoneSucceedsOnCleanBranchWithoutCommits(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)

	// Fresh worktree has 0 commits ahead of main and no upstream
	err := Done(DoneOptions{
		RepoRoot:     repoDir,
		WorktreePath: wtDir,
		LockPath:     lockPath,
		Force:        false,
	})
	if err != nil {
		t.Fatalf("expected Done on fresh clean worktree to succeed, got: %v", err)
	}

	if _, statErr := os.Stat(wtDir); !os.IsNotExist(statErr) {
		t.Errorf("expected worktree directory to be removed")
	}
}
