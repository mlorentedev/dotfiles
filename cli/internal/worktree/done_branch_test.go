package worktree

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func branchExists(repo, branch string) bool {
	return exec.Command("git", "-C", repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil
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
	tip := commitFile(t, wtDir, "work.txt")
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
	tip := commitFile(t, wtDir, "wip.txt")

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

// A pull-request lookup that fails cannot prove a squash landing, so the
// branch stays and the reason names the failure.
func TestDoneKeepsTheBranchWhenThePullRequestLookupFails(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	stubMergedPRHeads(t, nil, errors.New("gh: HTTP 502"))
	commitFile(t, wtDir, "wip.txt")

	res, err := Done(DoneOptions{RepoRoot: repoDir, WorktreePath: wtDir, LockPath: lockPath, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if !branchExists(repoDir, "feat/test") || res.DeletedBranch != "" || !strings.Contains(res.KeptReason, "HTTP 502") {
		t.Errorf("result %+v; want feat/test kept, naming the lookup failure", res)
	}
}
