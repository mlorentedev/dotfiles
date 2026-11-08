package lane

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

func runGitLaneTest(t *testing.T, dir string, args ...string) string {
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

func setupLaneTestRepo(t *testing.T) (repoDir, wtRoot string) {
	t.Helper()
	root := t.TempDir()
	originDir := filepath.Join(root, "origin.git")
	repoDir = filepath.Join(root, "dotfiles")
	wtRoot = filepath.Join(root, "worktrees")

	if err := os.MkdirAll(originDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitLaneTest(t, originDir, "init", "--bare", "-b", "main")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitLaneTest(t, repoDir, "init", "-b", "main")
	runGitLaneTest(t, repoDir, "remote", "add", "origin", originDir)
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitLaneTest(t, repoDir, "add", "README.md")
	runGitLaneTest(t, repoDir, "commit", "-m", "initial")
	runGitLaneTest(t, repoDir, "push", "-u", "origin", "main")
	return repoDir, wtRoot
}

func TestLaneInitIdempotent(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)

	changed, err := InitLanes(repoDir, wtRoot)
	if err != nil {
		t.Fatalf("first InitLanes failed: %v", err)
	}
	if changed != 3 {
		t.Fatalf("first InitLanes changed = %d, want 3", changed)
	}

	for _, name := range CanonicalLanes {
		lanePath := filepath.Join(wtRoot, "dotfiles", name)
		meta, err := worktree.LoadMetadata(lanePath)
		if err != nil || meta == nil {
			t.Fatalf("LoadMetadata(%s) failed: %v", lanePath, err)
		}
		if meta.Lane != name || meta.State != "PARKED" || meta.ReapOK {
			t.Errorf("lane %s metadata = %+v, want Lane=%s State=PARKED ReapOK=false", name, meta, name)
		}
	}

	// Second call must be idempotent (changed == 0).
	changed2, err := InitLanes(repoDir, wtRoot)
	if err != nil {
		t.Fatalf("second InitLanes failed: %v", err)
	}
	if changed2 != 0 {
		t.Fatalf("second InitLanes changed = %d, want 0 (idempotent)", changed2)
	}
}

func TestLaneSwitchRoutingAndWIPCap(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)
	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes: %v", err)
	}

	// 1. First feat/ branch routes to feat-lane-primary.
	lane1, path1, err := SwitchLane(repoDir, wtRoot, "feat/101-first", "")
	if err != nil {
		t.Fatalf("SwitchLane(feat/101-first) failed: %v", err)
	}
	if lane1 != LaneFeatPrimary {
		t.Errorf("lane1 = %q, want %q", lane1, LaneFeatPrimary)
	}
	meta1, err := worktree.LoadMetadata(path1)
	if err != nil || meta1 == nil {
		t.Fatalf("LoadMetadata(%s): %v", path1, err)
	}
	if meta1.State != "ACTIVE" || meta1.Issue != 101 {
		t.Errorf("meta1 = %+v, want State=ACTIVE Issue=101", meta1)
	}

	// 2. Second refactor/ branch routes to feat-lane-secondary.
	lane2, _, err := SwitchLane(repoDir, wtRoot, "refactor/102-second", "")
	if err != nil {
		t.Fatalf("SwitchLane(refactor/102-second) failed: %v", err)
	}
	if lane2 != LaneFeatSecondary {
		t.Errorf("lane2 = %q, want %q", lane2, LaneFeatSecondary)
	}

	// 3. Third feat/ branch must fail closed with WIP cap error (both feat lanes ACTIVE).
	if _, _, err := SwitchLane(repoDir, wtRoot, "feat/103-overflow", ""); err == nil {
		t.Fatalf("expected SwitchLane(feat/103-overflow) to fail closed when both feat lanes are ACTIVE")
	}

	// 4. fix/ branch routes to fast-track-lane even when both feat lanes are ACTIVE.
	lane3, _, err := SwitchLane(repoDir, wtRoot, "fix/104-urgent", "")
	if err != nil {
		t.Fatalf("SwitchLane(fix/104-urgent) failed: %v", err)
	}
	if lane3 != LaneFastTrack {
		t.Errorf("lane3 = %q, want %q", lane3, LaneFastTrack)
	}
}
