package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// A named lane sweep could not park is printed with its reason, next to the
// session and merge-unknown skips, so "skipped 1" is never the whole story
// (#3263). The lane's branch is landed by ancestry, so no forge is asked; a
// held index lock makes the detach fail while status still reads clean.
func TestWorktreeSweepPrintsWhyALaneWasNotParked(t *testing.T) {
	repo := worktreeTestRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.WriteFile(filepath.Join(repo, ".git", "info", "exclude"), []byte(worktree.MetadataFileName+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lane := filepath.Join(os.Getenv("WORKTREE_ROOT"), "repo", "lane-a")
	git := func(dir string, args ...string) string {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = dir
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git(repo, "worktree", "add", "-q", "-b", "feat/lane-a", lane)
	if err := os.WriteFile(filepath.Join(lane, "work.txt"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(lane, "add", "work.txt")
	git(lane, "-c", "user.name=t", "-c", "user.email=t@example.test", "commit", "-q", "-m", "work")
	git(repo, "merge", "-q", "--ff-only", "feat/lane-a")
	git(repo, "push", "-q", "origin", "main")
	now := time.Now()
	if err := worktree.SaveMetadata(lane, worktree.Metadata{
		Lane: "lane-a", State: "ACTIVE", CreatedAt: now.Add(-2 * time.Hour), LeaseExpiresAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	gitDir := git(lane, "rev-parse", "--absolute-git-dir")
	if err := os.WriteFile(filepath.Join(gitDir, "index.lock"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	out, err := executeWorktree(t, "sweep")
	if err != nil {
		t.Fatalf("sweep: %v\n%s", err, out)
	}
	// git names the lane by its physical path; macOS's $TMPDIR is a symlink.
	physical, err := filepath.EvalSymlinks(lane)
	if err != nil {
		t.Fatal(err)
	}
	want := "skipped: " + physical + ": git refused to detach it: "
	if !strings.Contains(out, want) || !strings.Contains(out, "index.lock") {
		t.Errorf("sweep output lacks %q with git's reason:\n%s", want, out)
	}
	if strings.Contains(out, "reaped 1") {
		t.Errorf("sweep reported the lane as parked:\n%s", out)
	}
}
