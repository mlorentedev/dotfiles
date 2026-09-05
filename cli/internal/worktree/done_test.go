package worktree

import (
	"path/filepath"
	"testing"
)

func TestDoneTeardown(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")
	wt := filepath.Join(tmpDir, "myrepo-wt-test")

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

	_ = wt
}
