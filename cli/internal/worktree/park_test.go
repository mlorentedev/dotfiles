package worktree

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
)

func runGitParkTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s failed: %v\n%s", args, dir, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func TestLaneParkInPlaceAndTipVerification(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

	// 1. Classify() behavior for persistent named lanes (B5, M2).
	parkedInfo := Info{
		Path:       "/wt/dotfiles/feat-lane-primary",
		IsDetached: true,
		Dirty:      false,
		Metadata: &Metadata{
			Lane:           "feat-lane-primary",
			State:          "PARKED",
			CreatedAt:      now.Add(-2 * time.Hour),
			LeaseExpiresAt: now.Add(-1 * time.Hour),
			ReapOK:         false,
		},
	}
	if st, _ := Classify(parkedInfo, now); st != StateParked {
		t.Fatalf("Classify(PARKED lane) = %q, want %q", st, StateParked)
	}

	parkableInfo := Info{
		Path:     "/wt/dotfiles/feat-lane-primary",
		Branch:   "feat/3255-lanes",
		Dirty:    false,
		PRMerged: true,
		Metadata: &Metadata{
			Lane:           "feat-lane-primary",
			State:          "ACTIVE",
			Issue:          3255,
			Persona:        "builder",
			CreatedAt:      now.Add(-2 * time.Hour),
			LeaseExpiresAt: now.Add(-1 * time.Minute),
			ReapOK:         false,
		},
	}
	if st, _ := Classify(parkableInfo, now); st != StateParkable {
		t.Fatalf("Classify(merged expired ACTIVE named lane) = %q, want %q", st, StateParkable)
	}

	// 2. Real git repo + linked named lane worktree for Park() & Done() delegation.
	root := t.TempDir()
	originDir := filepath.Join(root, "origin.git")
	repoDir := filepath.Join(root, "dotfiles")
	wtDir := filepath.Join(root, "worktrees", "dotfiles", "feat-lane-primary")

	if err := os.MkdirAll(originDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitParkTest(t, originDir, "init", "--bare", "-b", "main")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitParkTest(t, repoDir, "init", "-b", "main")
	runGitParkTest(t, repoDir, "remote", "add", "origin", originDir)
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitParkTest(t, repoDir, "add", "README.md")
	runGitParkTest(t, repoDir, "commit", "-m", "initial")
	baseSHA := runGitParkTest(t, repoDir, "rev-parse", "HEAD")
	runGitParkTest(t, repoDir, "push", "-u", "origin", "main")

	if err := os.MkdirAll(filepath.Dir(wtDir), 0o755); err != nil {
		t.Fatal(err)
	}
	runGitParkTest(t, repoDir, "worktree", "add", "--no-track", "-b", "feat/3255-lanes", wtDir, "origin/main")
	ensureExclude(repoDir, MetadataFileName)
	ensureExclude(repoDir, ".dotf-lane-prompt.md")

	// Commit on the feature branch.
	if err := os.WriteFile(filepath.Join(wtDir, "feature.txt"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitParkTest(t, wtDir, "add", "feature.txt")
	runGitParkTest(t, wtDir, "commit", "-m", "feat work")
	featTipSHA := runGitParkTest(t, wtDir, "rev-parse", "HEAD")

	meta := Metadata{
		Creator:        "claude",
		Issue:          3255,
		Lane:           "feat-lane-primary",
		State:          "ACTIVE",
		Persona:        "builder",
		CreatedAt:      now.Add(-2 * time.Hour),
		LeaseExpiresAt: now.Add(-1 * time.Hour),
		ReapOK:         false,
	}
	if err := SaveMetadata(wtDir, meta); err != nil {
		t.Fatal(err)
	}
	promptPath := filepath.Join(wtDir, ".dotf-lane-prompt.md")
	if err := os.WriteFile(promptPath, []byte("prompt content\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 2a. Dirty worktree must fail closed.
	dirtyFile := filepath.Join(wtDir, "dirty.txt")
	if err := os.WriteFile(dirtyFile, []byte("uncommitted\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Park(DoneOptions{
		RepoRoot:            repoDir,
		WorktreePath:        wtDir,
		LockPath:            filepath.Join(root, "wt.lock"),
		IgnoreRecentSession: true,
		LandedHeads: func(_, _ string) ([]string, error) {
			return []string{featTipSHA}, nil
		},
	})
	if err == nil {
		t.Fatalf("expected Park() to fail on dirty worktree, got nil")
	}
	_ = os.Remove(dirtyFile)

	// 2b. Unpushed local commit after PR squash-merge (tip not in LandedHeads) must fail closed (M3).
	_, err = Park(DoneOptions{
		RepoRoot:            repoDir,
		WorktreePath:        wtDir,
		LockPath:            filepath.Join(root, "wt.lock"),
		IgnoreRecentSession: true,
		LandedHeads: func(_, _ string) ([]string, error) {
			// Pretend an older commit merged, not featTipSHA.
			return []string{baseSHA}, nil
		},
	})
	if err == nil {
		t.Fatalf("expected Park() to refuse when current branch tip SHA is not in merged PR heads (M3)")
	}

	// 2c. Done() on a named lane delegates to Park() when tip SHA is verified landed.
	res, err := Done(DoneOptions{
		RepoRoot:            repoDir,
		WorktreePath:        wtDir,
		LockPath:            filepath.Join(root, "wt.lock"),
		IgnoreRecentSession: true,
		LandedHeads: func(_, _ string) ([]string, error) {
			return []string{featTipSHA}, nil
		},
	})
	if err != nil {
		t.Fatalf("Done() delegating to Park() failed: %v", err)
	}
	if res.DeletedBranch != "feat/3255-lanes" {
		t.Errorf("DeletedBranch = %q, want %q", res.DeletedBranch, "feat/3255-lanes")
	}
	if res.DeletedSHA != featTipSHA || len(res.DeletedSHA) != 40 {
		t.Errorf("DeletedSHA = %q, want 40-char %q", res.DeletedSHA, featTipSHA)
	}

	// Directory MUST still exist on disk (in-place park, never git worktree remove).
	if fi, err := os.Stat(wtDir); err != nil || !fi.IsDir() {
		t.Fatalf("named lane directory %s was removed or missing: %v", wtDir, err)
	}

	// Must be detached at origin/main.
	headAfter := runGitParkTest(t, wtDir, "rev-parse", "HEAD")
	if headAfter != baseSHA {
		t.Errorf("HEAD after Park = %q, want baseSHA %q", headAfter, baseSHA)
	}
	if _, err := os.Stat(promptPath); !os.IsNotExist(err) {
		t.Errorf(".dotf-lane-prompt.md should be removed on Park (M4), stat err = %v", err)
	}

	afterMeta, err := LoadMetadata(wtDir)
	if err != nil || afterMeta == nil {
		t.Fatalf("LoadMetadata after Park failed: %v", err)
	}
	if afterMeta.State != "PARKED" || afterMeta.Lane != "feat-lane-primary" || afterMeta.Issue != 0 || afterMeta.Persona != "" || afterMeta.ReapOK {
		t.Errorf("unexpected metadata after Park: %+v", afterMeta)
	}

	// 3. SessionEnd must be a clean no-op on a PARKED lane (B3), and ArchiveAndPruneThread moves thread to archive.
	vaultDir := filepath.Join(root, "vault")
	memDir := filepath.Join(vaultDir, "10_projects", "dotfiles", "memory")
	if err := os.MkdirAll(filepath.Join(memDir, "threads"), 0o755); err != nil {
		t.Fatal(err)
	}
	memPath := filepath.Join(memDir, "MEMORY.md")
	initialMem := "# Dotfiles Memory\n\n## Session Handoff\n\n### thread: feat-3255-lanes (writer: claude)\n\n> Updated: 2026-11-08<br>\n**Next action:** Continue<br>\n[full thread](threads/feat-3255-lanes.md)\n"
	if err := os.WriteFile(memPath, []byte(initialMem), 0o644); err != nil {
		t.Fatal(err)
	}
	threadFile := filepath.Join(memDir, "threads", "feat-3255-lanes.md")
	if err := os.WriteFile(threadFile, []byte("### thread: feat-3255-lanes (writer: claude)\n\n> Updated: 2026-11-08\n**Last task:** Done\n**Next action:** None\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(map[string]string{
		"cwd":        wtDir,
		"session_id": "test-session",
	})
	journalOut, err := mem.SessionEnd(payload, vaultDir, now)
	if err != nil {
		t.Fatalf("SessionEnd on PARKED lane returned error: %v", err)
	}
	if journalOut != "" {
		t.Errorf("SessionEnd on PARKED lane wrote journal %q, want no-op empty string (B3)", journalOut)
	}

	if err := mem.ArchiveAndPruneThread(filepath.Join(vaultDir, "10_projects", "dotfiles"), "feat/3255-lanes"); err != nil {
		t.Fatalf("ArchiveAndPruneThread failed: %v", err)
	}
	if _, err := os.Stat(threadFile); !os.IsNotExist(err) {
		t.Errorf("expected active thread file %s to be moved out of threads/", threadFile)
	}
	archivedThread := filepath.Join(memDir, "archive", "feat-3255-lanes.md")
	if _, err := os.Stat(archivedThread); err != nil {
		t.Errorf("expected archived thread file at %s, stat err: %v", archivedThread, err)
	}
	updatedMem, err := os.ReadFile(memPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updatedMem), "feat-3255-lanes") {
		t.Errorf("expected MEMORY.md to have pruned feat-3255-lanes entry, got:\n%s", string(updatedMem))
	}
}
