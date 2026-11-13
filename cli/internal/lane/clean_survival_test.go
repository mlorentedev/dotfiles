package lane

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// The defect behind ADR-062: `git clean -fdx` in an active lane removed its
// in-tree lease, so status read the lane as leaseless and the pre-commit guard
// refused the next commit. With the lease in the worktree's git admin dir the
// clean changes nothing a reader of the lease can see.
func TestActiveLaneSurvivesGitClean(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)
	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes failed: %v", err)
	}
	laneName, lanePath, err := SwitchLane(repoDir, wtRoot, "feat/594-clean-survival", "")
	if err != nil {
		t.Fatalf("SwitchLane failed: %v", err)
	}

	runGitLaneTest(t, lanePath, "clean", "-fdx")
	if _, err := os.Stat(filepath.Join(lanePath, worktree.MetadataFileName)); !os.IsNotExist(err) {
		t.Fatalf("the in-tree mirror survived `git clean -fdx` (err=%v): the test no longer exercises the defect", err)
	}

	meta, err := worktree.LoadMetadata(lanePath)
	if err != nil || meta == nil {
		t.Fatalf("LoadMetadata after clean = %v, %v; want the lease", meta, err)
	}
	if meta.Lane != laneName || !strings.EqualFold(meta.State, string(worktree.StateActive)) {
		t.Errorf("lease after clean = %+v; want lane %s ACTIVE", meta, laneName)
	}
	infos, err := worktree.List(repoDir, func(string, string) ([]string, error) { return nil, nil })
	if err != nil {
		t.Fatal(err)
	}
	var listed *worktree.Info
	for i := range infos {
		if samePath(infos[i].Path, lanePath) {
			listed = &infos[i]
		}
	}
	if listed == nil {
		t.Fatalf("lane %s not in worktree list %+v", lanePath, infos)
	}
	if listed.State != worktree.StateActive || listed.LeaseLegacyOnly {
		t.Errorf("worktree list after clean: state=%s legacy=%v reason=%q; want ACTIVE from the admin-dir lease", listed.State, listed.LeaseLegacyOnly, listed.StateReason)
	}

	// A commit through the installed guard: the hook reads the same lease.
	guard, err := filepath.Abs(filepath.Join("..", "..", "..", "git-hooks", "lib", "lane-guard.sh"))
	if err != nil {
		t.Fatal(err)
	}
	hooks := t.TempDir()
	hook := "#!/bin/sh\nexec sh \"" + filepath.ToSlash(guard) + "\"\n"
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte(hook), 0o755); err != nil { // #nosec G306 -- a hook must be executable
		t.Fatal(err)
	}
	runGitLaneTest(t, lanePath, "config", "core.hooksPath", hooks)
	if err := os.WriteFile(filepath.Join(lanePath, "work.txt"), []byte("after clean\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitLaneTest(t, lanePath, "add", "work.txt")
	runGitLaneTest(t, lanePath, "commit", "-m", "work after clean")
}
