package lane

import (
	"os"
	"path/filepath"
	"testing"
)

// A lane's identity is read from the worktree's own git admin dir, which git
// creates, moves and prunes with the worktree. Nothing about the path is
// stored, so a lane keeps its name after `git worktree move` and a directory
// that merely shares a lane's name has none.
func TestIdentity_ReadsTheRecordFromTheWorktreeGitDir(t *testing.T) {
	repo, wtRoot := setupLaneTestRepo(t)
	linked := filepath.Join(wtRoot, "any-leaf")
	runGitLaneTest(t, repo, "worktree", "add", "--detach", linked, "HEAD")
	gitDir := runGitLaneTest(t, linked, "rev-parse", "--absolute-git-dir")
	if err := os.WriteFile(filepath.Join(gitDir, LaneIdentityFile), []byte(LaneFeatPrimary+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	name, ok, err := Identity(linked)
	if err != nil || !ok || name != LaneFeatPrimary {
		t.Fatalf("Identity(linked) = %q, %v, %v; want %q, true, nil", name, ok, err, LaneFeatPrimary)
	}

	moved := filepath.Join(wtRoot, "another-leaf")
	runGitLaneTest(t, repo, "worktree", "move", linked, moved)
	if name, ok, err := Identity(moved); err != nil || !ok || name != LaneFeatPrimary {
		t.Fatalf("Identity(moved) = %q, %v, %v; want the record to survive the move", name, ok, err)
	}

	bare := filepath.Join(wtRoot, "no-record")
	runGitLaneTest(t, repo, "worktree", "add", "--detach", bare, "HEAD")
	// A main checkout's git dir is the common dir, where `dotf lane next` keeps
	// its prompt DIRECTORY under the same name (cli/internal/cmd/lane.go). The
	// record is a worktree's, so the main checkout is never a lane, prompt dir
	// or not.
	if err := os.MkdirAll(filepath.Join(repo, ".git", LaneIdentityFile), 0o755); err != nil {
		t.Fatal(err)
	}
	for what, path := range map[string]string{
		"a worktree without a record":                     bare,
		"the main checkout carrying the prompt directory": repo,
		"a directory outside git":                         t.TempDir(),
	} {
		if name, ok, err := Identity(path); err != nil || ok || name != "" {
			t.Errorf("Identity(%s) = %q, %v, %v; want \"\", false, nil", what, name, ok, err)
		}
	}
}

func TestWriteIdentity_IsIdempotent(t *testing.T) {
	repo, wtRoot := setupLaneTestRepo(t)
	linked := filepath.Join(wtRoot, "lane")
	runGitLaneTest(t, repo, "worktree", "add", "--detach", linked, "HEAD")

	changed, err := writeIdentity(linked, LaneFastTrack)
	if err != nil || !changed {
		t.Fatalf("first writeIdentity = %v, %v; want true, nil", changed, err)
	}
	if changed, err := writeIdentity(linked, LaneFastTrack); err != nil || changed {
		t.Fatalf("second writeIdentity = %v, %v; want false, nil", changed, err)
	}
	if name, ok, _ := Identity(linked); !ok || name != LaneFastTrack {
		t.Fatalf("Identity after write = %q, %v", name, ok)
	}
	if _, err := writeIdentity(repo, LaneFastTrack); err == nil {
		t.Fatal("writeIdentity on the main checkout must refuse: it has no worktree admin dir")
	}
}
