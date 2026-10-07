package worktree

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// #1608: a squash-merged branch whose remote GitHub deleted is ahead of the
// base forever. Whether its work landed is a question for the merged pull
// request's head, answered here by a stub instead of GitHub.

func stubMergedPRHeads(t *testing.T, heads []string, err error) {
	t.Helper()
	mergedPRHeads = func(string, string) ([]string, error) { return heads, err }
	t.Cleanup(func() { mergedPRHeads = ghMergedPRHeads })
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func commitFile(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOut(t, dir, "add", name)
	gitOut(t, dir, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "-m", name)
	return gitOut(t, dir, "rev-parse", "HEAD")
}

func done(t *testing.T, repoDir, wtDir, lockPath string) error {
	t.Helper()
	return Done(DoneOptions{RepoRoot: repoDir, WorktreePath: wtDir, LockPath: lockPath})
}

func TestDoneRemovesASquashMergedBranchWhoseHeadIsTheMergedHead(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	tip := commitFile(t, wtDir, "work.txt")
	stubMergedPRHeads(t, []string{tip}, nil)

	if err := done(t, repoDir, wtDir, lockPath); err != nil {
		t.Fatalf("a branch whose tip is the merged PR's head must be removable: %v", err)
	}
}

// The PR took a push from elsewhere (a rebase, a merge of main) after this
// checkout last pulled: the checkout is behind the merged head, not ahead.
func TestDoneRemovesABranchBehindItsMergedHead(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	commitFile(t, wtDir, "work.txt")
	gitOut(t, wtDir, "branch", "later")
	gitOut(t, wtDir, "checkout", "-q", "later")
	head := commitFile(t, wtDir, "pushed-elsewhere.txt")
	gitOut(t, wtDir, "checkout", "-q", "feat/test")
	stubMergedPRHeads(t, []string{"0000000000000000000000000000000000000000", head}, nil)

	if err := done(t, repoDir, wtDir, lockPath); err != nil {
		t.Fatalf("a checkout contained in the merged head has nothing unpushed: %v", err)
	}
}

func TestDoneRefusesACommitMadeAfterTheMerge(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	merged := commitFile(t, wtDir, "work.txt")
	commitFile(t, wtDir, "follow-up.txt")
	stubMergedPRHeads(t, []string{merged}, nil)

	err := done(t, repoDir, wtDir, lockPath)
	if err == nil || !strings.Contains(err.Error(), "unpushed commit") {
		t.Fatalf("a commit in no merged head is unpushed work, got %v", err)
	}
	if _, statErr := os.Stat(wtDir); statErr != nil {
		t.Errorf("the worktree was removed with unpushed work: %v", statErr)
	}
}

// No gh, offline, rate-limited: the question cannot be answered, so it fails
// closed and says why.
func TestDoneRefusesWhenMergedPullRequestsCannotBeListed(t *testing.T) {
	repoDir, wtDir, lockPath := setupTestGitRepoAndWorktree(t)
	commitFile(t, wtDir, "work.txt")
	stubMergedPRHeads(t, nil, errors.New("gh: not found"))

	err := done(t, repoDir, wtDir, lockPath)
	if err == nil || !strings.Contains(err.Error(), "could not check for a merged pull request: gh: not found") {
		t.Fatalf("want a refusal naming the failed check, got %v", err)
	}
}

// list and sweep ask the same question by branch name: a merged PR does not
// make a branch with later commits merged (#1608's related hazard).
func TestIsPRMergedRequiresTheBranchTipInAMergedHead(t *testing.T) {
	repoDir, wtDir, _ := setupTestGitRepoAndWorktree(t)
	merged := commitFile(t, wtDir, "work.txt")

	stubMergedPRHeads(t, []string{merged}, nil)
	if got, _ := (&RealGitRunner{}).IsPRMerged(repoDir, "feat/test"); !got {
		t.Error("a tip that is the merged head must read as merged")
	}

	commitFile(t, wtDir, "follow-up.txt")
	if got, _ := (&RealGitRunner{}).IsPRMerged(repoDir, "feat/test"); got {
		t.Error("a tip past the merged head must not read as merged")
	}
}
