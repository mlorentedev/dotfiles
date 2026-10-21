package worktree

import (
	"errors"
	"strings"
	"testing"
)

// #2818: list and sweep ask the forge reader the caller hands them, as done
// does (#2759), so a squash merge on Gitea reads as landed. gh, which reaches
// GitHub only, is never asked.

func featInfo(t *testing.T, infos []Info) Info {
	t.Helper()
	for _, info := range infos {
		if strings.HasSuffix(info.Branch, "feat/test") {
			return info
		}
	}
	t.Fatalf("no feat/test worktree in %+v", infos)
	return Info{}
}

func TestListAsksTheLandedHeadsTheCallerHandsIt(t *testing.T) {
	repoDir, wtDir, _ := setupTestGitRepoAndWorktree(t)
	tip := commitFile(t, wtDir, "work.txt")
	stubMergedPRHeads(t, nil, errors.New("gh must not be asked"))
	infos, err := List(repoDir, func(_, branch string) ([]string, error) {
		if !strings.HasSuffix(branch, "feat/test") {
			return nil, nil
		}
		return []string{tip}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !featInfo(t, infos).PRMerged {
		t.Error("a branch the forge says landed was listed as not merged")
	}
}

// A forge that cannot answer must never make a branch read as merged: that is
// the direction in which sweep would reap a worktree whose work did not land.
func TestListReadsAForgeErrorAsNotMerged(t *testing.T) {
	repoDir, wtDir, _ := setupTestGitRepoAndWorktree(t)
	commitFile(t, wtDir, "work.txt")
	infos, err := List(repoDir, func(string, string) ([]string, error) {
		return nil, errors.New("no forge backend for gitea.example")
	})
	if err != nil {
		t.Fatal(err)
	}
	if featInfo(t, infos).PRMerged {
		t.Error("a forge error was read as merged")
	}
}

func TestSweepRunnerAsksTheLandedHeadsInItsOptions(t *testing.T) {
	repoDir, wtDir, _ := setupTestGitRepoAndWorktree(t)
	tip := commitFile(t, wtDir, "work.txt")
	stubMergedPRHeads(t, nil, errors.New("gh must not be asked"))
	runner := realSweepRunner(SweepOptions{RepoRoot: repoDir,
		LandedHeads: func(string, string) ([]string, error) { return []string{tip}, nil }})
	if merged, err := runner.IsPRMerged(repoDir, "feat/test"); err != nil || !merged {
		t.Errorf("IsPRMerged = %v, %v: want the forge's answer, merged", merged, err)
	}
}
