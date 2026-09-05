package worktree

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListWorktrees(t *testing.T) {
	// Create a temporary mock directory structure
	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")
	wt1 := filepath.Join(tmpDir, "myrepo-wt-feat1")
	wt2 := filepath.Join(tmpDir, "myrepo-wt-feat2")
	submodule := filepath.Join(mainRepo, "vendor", "submod")

	_ = os.MkdirAll(mainRepo, 0o755)
	_ = os.MkdirAll(wt1, 0o755)
	_ = os.MkdirAll(wt2, 0o755)
	_ = os.MkdirAll(submodule, 0o755)

	// Write metadata for wt1 (expired lease, reap_ok true)
	meta1 := Metadata{
		Creator:        "test-agent",
		Issue:          123,
		CreatedAt:      time.Now().Add(-2 * time.Hour),
		LeaseExpiresAt: time.Now().Add(-1 * time.Hour),
		ReapOK:         true,
	}
	_ = SaveMetadata(wt1, meta1)

	// Mock porcelain output
	mockPorcelain := "worktree " + mainRepo + "\n" +
		"HEAD 1111\n" +
		"branch refs/heads/main\n\n" +
		"worktree " + wt1 + "\n" +
		"HEAD 2222\n" +
		"branch refs/heads/feat/feat1\n\n" +
		"worktree " + wt2 + "\n" +
		"HEAD 3333\n" +
		"branch refs/heads/feat/feat2\n\n" +
		"worktree " + submodule + "\n" +
		"HEAD 4444\n" +
		"branch refs/heads/main\n" +
		"gitdir " + filepath.Join(mainRepo, ".git", "modules", "submod") + "\n"

	runner := &MockGitRunner{
		PorcelainOutput: mockPorcelain,
		DirtyPaths:      map[string]bool{wt2: true},
		MergedBranches:  map[string]bool{"feat/feat1": true},
	}

	infos, err := ListWithRunner(mainRepo, runner, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Submodule should be ignored; so exactly 3 worktrees (main, wt1, wt2)
	if len(infos) != 3 {
		t.Fatalf("expected 3 worktrees (submodule filtered out), got %d", len(infos))
	}

	// Verify main
	if !infos[0].IsMain || infos[0].State != StateActive {
		t.Errorf("expected infos[0] to be main active, got %v", infos[0])
	}

	// Verify wt1: clean, merged, expired lease -> REAPABLE
	if infos[1].Path != wt1 || infos[1].State != StateReapable {
		t.Errorf("expected infos[1] to be REAPABLE, got %s (reason: %s)", infos[1].State, infos[1].StateReason)
	}

	// Verify wt2: dirty -> DIRTY
	if infos[2].Path != wt2 || infos[2].State != StateDirty {
		t.Errorf("expected infos[2] to be DIRTY, got %s", infos[2].State)
	}
}

func TestParseGitHubSlug(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"git@github.com:mlorentedev/dotfiles.git", "mlorentedev/dotfiles"},
		{"https://github.com/mlorentedev/dotfiles.git", "mlorentedev/dotfiles"},
		{"https://github.com/mlorentedev/dotfiles", "mlorentedev/dotfiles"},
		{"git@github.com:org/repo", "org/repo"},
		{"ssh://git@github.com/org/repo.git", "org/repo"},
		{"https://gitlab.com/org/repo.git", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := ParseGitHubSlug(tc.input)
		if got != tc.want {
			t.Errorf("ParseGitHubSlug(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestListWithRunnerDirtyErrorFailsClosed(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")
	wt1 := filepath.Join(tmpDir, "myrepo-wt-err")
	_ = os.MkdirAll(mainRepo, 0o755)
	_ = os.MkdirAll(wt1, 0o755)

	meta := Metadata{
		Creator:        "test-agent",
		Issue:          123,
		CreatedAt:      time.Now().Add(-2 * time.Hour),
		LeaseExpiresAt: time.Now().Add(-1 * time.Hour),
		ReapOK:         true,
	}
	_ = SaveMetadata(wt1, meta)

	mockPorcelain := "worktree " + mainRepo + "\n" +
		"HEAD 1111\n" +
		"branch refs/heads/main\n\n" +
		"worktree " + wt1 + "\n" +
		"HEAD 2222\n" +
		"branch refs/heads/feat/err\n\n"

	runner := &MockGitRunner{
		PorcelainOutput: mockPorcelain,
		DirtyErrors:     map[string]error{wt1: os.ErrPermission},
		MergedBranches:  map[string]bool{"feat/err": true},
	}

	infos, err := ListWithRunner(mainRepo, runner, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(infos) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(infos))
	}

	// Because IsDirty errored, wt1 must be treated as dirty (fail-closed)
	if !infos[1].Dirty || infos[1].State != StateDirty {
		t.Errorf("expected wt1 with IsDirty error to be classified StateDirty, got %s (dirty=%v)", infos[1].State, infos[1].Dirty)
	}
}
