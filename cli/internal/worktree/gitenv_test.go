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

// The fetch sets one variable of its own; adding it to os.Environ() instead of
// gitIn's environment put GIT_DIR back on the one call that reaches the network.
func TestFetchCommitCmd_KeepsTheScrub(t *testing.T) {
	t.Setenv("GIT_DIR", "/elsewhere/.git")
	env := strings.Join(fetchCommitCmd(t.TempDir(), "abc").Env, "\n")
	if strings.Contains(env, "GIT_DIR=") {
		t.Errorf("fetch inherits GIT_DIR:\n%s", env)
	}
	if !strings.Contains(env, "GIT_TERMINAL_PROMPT=0") {
		t.Errorf("fetch lost GIT_TERMINAL_PROMPT=0")
	}
}

// resolveBaseRef is called by the runner's merge check; its ref lookup must read
// the named repository too.
func TestFirstRef_ReadsTheNamedRepositoryWhateverGitDirSays(t *testing.T) {
	tmp := t.TempDir()
	mine, other := filepath.Join(tmp, "mine"), filepath.Join(tmp, "other")
	gitT(t, tmp, "init", "-q", "-b", "main", mine)
	gitT(t, tmp, "init", "-q", "-b", "trunk", other)
	for _, repo := range []string{mine, other} {
		gitT(t, repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	}
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	if got := firstRef(mine, []string{"main"}); got != "main" {
		t.Errorf("firstRef = %q, want main from the named repository", got)
	}
}
