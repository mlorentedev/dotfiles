package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func branchExists(repo, branch string) bool {
	return exec.Command("git", "-C", repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil
}

func commitIn(t *testing.T, wt, name string) string {
	t.Helper()
	_ = os.WriteFile(filepath.Join(wt, name), []byte(name), 0o644)
	gitOut(t, wt, "add", name)
	gitOut(t, wt, "commit", "-q", "-m", name)
	return gitOut(t, wt, "rev-parse", "HEAD")
}

func TestDoneDeletesALandedBranch(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	stubMergedPRHeads(t, nil, nil)
	tip := gitOut(t, wtDir, "rev-parse", "HEAD") // no commits of its own: an ancestor of main

	res, err := Done(DoneOptions{RepoRoot: repoDir, WorktreePath: wtDir, LockPath: lockPath})
	if err != nil {
		t.Fatal(err)
	}
	if branchExists(repoDir, "feat/test") || res.DeletedBranch != "feat/test" || res.DeletedSHA != tip {
		t.Errorf("result %+v; want feat/test deleted at %s", res, tip)
	}
}

func TestDoneDeletesASquashLandedBranch(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	tip := commitIn(t, wtDir, "work.txt")
	stubMergedPRHeads(t, []string{tip}, nil)

	res, err := Done(DoneOptions{RepoRoot: repoDir, WorktreePath: wtDir, LockPath: lockPath})
	if err != nil {
		t.Fatal(err)
	}
	if branchExists(repoDir, "feat/test") || res.DeletedSHA != tip {
		t.Errorf("result %+v; want the squash-landed branch deleted at %s", res, tip)
	}
}

// --force removes the worktree of unlanded work, and the branch is all that is
// left of that work: it stays.
func TestDoneKeepsAnUnlandedBranchEvenWithForce(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	stubMergedPRHeads(t, nil, nil)
	tip := commitIn(t, wtDir, "wip.txt")

	res, err := Done(DoneOptions{RepoRoot: repoDir, WorktreePath: wtDir, LockPath: lockPath, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if !branchExists(repoDir, "feat/test") || res.DeletedBranch != "" || res.KeptReason == "" {
		t.Errorf("result %+v; want feat/test kept with a reason", res)
	}
	if got := gitOut(t, repoDir, "rev-parse", "feat/test"); got != tip {
		t.Errorf("feat/test moved to %s, want %s", got, tip)
	}
}
