package worktree

import (
	"path/filepath"
	"strings"
	"testing"
)

// A GIT_DIR inherited from a hook or a parent git outranks -C, so without the
// scrub the runner lists the other repository's worktrees (#3122).
func TestRealGitRunner_ListsTheNamedRepositoryWhateverGitDirSays(t *testing.T) {
	tmp := t.TempDir()
	mine, other := filepath.Join(tmp, "mine"), filepath.Join(tmp, "other")
	for _, repo := range []string{mine, other} {
		gitT(t, tmp, "init", "-q", "-b", "main", repo)
		gitT(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	}
	gitT(t, other, "worktree", "add", "-q", "-b", "elsewhere", filepath.Join(tmp, "other-wt"))
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)

	got, err := (&RealGitRunner{}).WorktreeListPorcelain(mine)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "elsewhere") || strings.Contains(got, "other") {
		t.Errorf("listed another repository's worktrees:\n%s", got)
	}
	if !strings.Contains(got, "branch refs/heads/main") {
		t.Errorf("did not list the named repository:\n%s", got)
	}
}

func TestWithoutRepositoryEnv_KeepsEverythingElse(t *testing.T) {
	got := withoutRepositoryEnv([]string{"GIT_DIR=/x", "git_work_tree=/y", "GIT_CONFIG_GLOBAL=/g", "HOME=/h"})
	if strings.Join(got, " ") != "GIT_CONFIG_GLOBAL=/g HOME=/h" {
		t.Errorf("got %v", got)
	}
}
