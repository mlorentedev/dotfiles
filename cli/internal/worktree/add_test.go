package worktree

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveSiblingPath(t *testing.T) {
	cases := []struct {
		repoRoot string
		slug     string
		expected string
	}{
		{"/home/user/Projects/dotfiles", "auth-fix", "/home/user/Projects/dotfiles-wt-auth-fix"},
		{"/home/user/Projects/kubelab", "sec012", "/home/user/Projects/kubelab-wt-sec012"},
		{"/var/repo", "bug-1", "/var/repo-wt-bug-1"},
	}

	for _, tc := range cases {
		got := ResolveSiblingPath(tc.repoRoot, tc.slug)
		if got != tc.expected {
			t.Errorf("ResolveSiblingPath(%q, %q) = %q, expected %q", tc.repoRoot, tc.slug, got, tc.expected)
		}
	}
}

func TestValidateIsolation(t *testing.T) {
	repoRoot := "/home/user/Projects/dotfiles"

	// External sibling: OK
	if err := ValidateIsolation(repoRoot, "/home/user/Projects/dotfiles-wt-feat"); err != nil {
		t.Errorf("expected external sibling to be valid, got: %v", err)
	}

	// Completely distinct external path: OK
	if err := ValidateIsolation(repoRoot, "/tmp/some-wt"); err != nil {
		t.Errorf("expected distinct external path to be valid, got: %v", err)
	}

	// Nested inside repoRoot: REJECT
	if err := ValidateIsolation(repoRoot, "/home/user/Projects/dotfiles/wt-feat"); err == nil {
		t.Errorf("expected nested path inside repoRoot to be rejected")
	}

	// Nested inside a subdirectory: REJECT
	if err := ValidateIsolation(repoRoot, "/home/user/Projects/dotfiles/subdir/wt"); err == nil {
		t.Errorf("expected deep nested path inside repoRoot to be rejected")
	}

	// Same as repoRoot: REJECT
	if err := ValidateIsolation(repoRoot, repoRoot); err == nil {
		t.Errorf("expected repoRoot path itself to be rejected")
	}
}

func TestAddWorktreeWithRunner(t *testing.T) {
	tmpDir := t.TempDir()
	mainRepo := filepath.Join(tmpDir, "myrepo")
	_ = os.MkdirAll(filepath.Join(mainRepo, ".git", "info"), 0o755)

	targetPath := filepath.Join(tmpDir, "myrepo-wt-test")

	executedCommands := []string{}
	runner := &MockAddRunner{
		OnWorktreeAdd: func(repo, target, branch, base string) error {
			executedCommands = append(executedCommands, target)
			// Simulate git creating target directory
			_ = os.MkdirAll(target, 0o755)
			return nil
		},
	}

	opts := AddOptions{
		RepoRoot: mainRepo,
		Slug:     "test",
		Issue:    1500,
		TTL:      24 * time.Hour,
		Creator:  "test-agent",
	}

	info, err := AddWithRunner(opts, runner, time.Now())
	if err != nil {
		t.Fatalf("unexpected error adding worktree: %v", err)
	}

	if info.Path != targetPath {
		t.Errorf("expected info.Path to be %s, got %s", targetPath, info.Path)
	}

	// Verify .dotf-worktree.json was written
	meta, err := LoadMetadata(targetPath)
	if err != nil || meta == nil {
		t.Fatalf("failed to load written metadata: %v", err)
	}
	if meta.Issue != 1500 {
		t.Errorf("expected issue 1500, got %d", meta.Issue)
	}
	if meta.Creator != "test-agent" {
		t.Errorf("expected creator test-agent, got %s", meta.Creator)
	}
	if !meta.ReapOK {
		t.Errorf("expected reap_ok to be true by default")
	}

	// Verify .git/info/exclude in main repo has the metadata file
	excludeContent, _ := os.ReadFile(filepath.Join(mainRepo, ".git", "info", "exclude"))
	if !containsLine(string(excludeContent), MetadataFileName) {
		t.Errorf("expected %s to be added to .git/info/exclude", MetadataFileName)
	}
}

type MockAddRunner struct {
	OnWorktreeAdd func(repo, target, branch, base string) error
}

func (m *MockAddRunner) WorktreeAdd(repo, target, branch, base string) error {
	if m.OnWorktreeAdd != nil {
		return m.OnWorktreeAdd(repo, target, branch, base)
	}
	return nil
}

func TestCheckAutoCommitHooks(t *testing.T) {
	tmpDir := t.TempDir()
	hooksDir := filepath.Join(tmpDir, ".git", "hooks")
	_ = os.MkdirAll(hooksDir, 0o755)

	// Clean hooks directory: no warning
	if warn := CheckAutoCommitHooks(tmpDir); warn != "" {
		t.Errorf("expected no warning for clean repo, got: %s", warn)
	}

	// Add post-commit hook with auto-commit pattern
	postCommit := filepath.Join(hooksDir, "post-commit")
	_ = os.WriteFile(postCommit, []byte("#!/bin/sh\ngit commit -m 'auto-commit'\n"), 0o755)

	warn := CheckAutoCommitHooks(tmpDir)
	if warn == "" {
		t.Errorf("expected warning when auto-commit hook exists")
	}
}
