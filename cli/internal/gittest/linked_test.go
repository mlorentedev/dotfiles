package gittest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fake carries exactly what a reader of the .git pointer needs: the
// pointer, and the admin dir it names under the repository's worktrees/.
func TestLinkWorktreeWritesThePointerAndTheAdminDir(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	dir := filepath.Join(tmp, "wt")
	admin, err := LinkWorktree(repo, dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(repo, ".git", "worktrees", "wt"); admin != want {
		t.Fatalf("admin dir %q, want %q", admin, want)
	}
	if fi, err := os.Stat(admin); err != nil || !fi.IsDir() {
		t.Fatalf("admin dir not created: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(raw)); got != "gitdir: "+admin {
		t.Fatalf("pointer %q, want %q", got, "gitdir: "+admin)
	}
}
