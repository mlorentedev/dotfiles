package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// #2873: a forge that cannot answer keeps the verdict "not merged", and says
// why, so a squash-merged branch on an undeclared forge is not a silent
// UNMERGED.

func TestListSaysWhyTheForgeCouldNotAnswer(t *testing.T) {
	repoDir, wtDir, _ := setupTestGitRepoAndWorktree(t)
	commitFile(t, wtDir, "work.txt")
	infos, err := List(repoDir, func(string, string) ([]string, error) {
		return nil, errors.New("no forge backend for gitea.example")
	})
	if err != nil {
		t.Fatal(err)
	}
	info := featInfo(t, infos)
	if info.PRMerged || !strings.Contains(info.PRMergedUnknown, "gitea.example") {
		t.Errorf("PRMerged=%v PRMergedUnknown=%q: want not merged, with the forge named", info.PRMerged, info.PRMergedUnknown)
	}
}

func TestClassifyNamesAnUnansweredForgeAsTheReason(t *testing.T) {
	now := time.Now()
	info := Info{Branch: "feat/x", PRMergedUnknown: "no forge backend for gitea.example",
		Metadata: &Metadata{ReapOK: true, CreatedAt: now.Add(-2 * time.Hour), LeaseExpiresAt: now.Add(-time.Hour)}}
	state, reason := Classify(info, now)
	if state != StateUnmerged || !strings.Contains(reason, "gitea.example") {
		t.Errorf("Classify = %s %q: want unmerged, naming the forge", state, reason)
	}
}

func TestSweepReportsAWorktreeKeptBecauseTheForgeCouldNotAnswer(t *testing.T) {
	withHostProcessInside(t, false)
	tmpDir := t.TempDir()
	mainRepo, wt := filepath.Join(tmpDir, "myrepo"), filepath.Join(tmpDir, "myrepo-wt-x")
	for _, d := range []string{mainRepo, wt} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	if err := SaveMetadata(wt, Metadata{ReapOK: true, CreatedAt: now.Add(-2 * time.Hour), LeaseExpiresAt: now.Add(-time.Hour)}); err != nil {
		t.Fatal(err)
	}
	runner := &MockSweepRunner{MockGitRunner: MockGitRunner{
		PorcelainOutput: "worktree " + mainRepo + "\nHEAD 1111\nbranch refs/heads/main\n\n" +
			"worktree " + wt + "\nHEAD 2222\nbranch refs/heads/feat/x\n\n",
		MergeErrors: map[string]error{"feat/x": errors.New("no forge backend for gitea.example")},
	}}
	report, err := SweepWithRunner(SweepOptions{RepoRoot: mainRepo, LockPath: filepath.Join(tmpDir, "sweep.lock")}, runner, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Reaped) != 0 {
		t.Fatalf("reaped %d worktree(s) the forge could not confirm as merged", len(report.Reaped))
	}
	if len(report.MergeUnknownSkips) != 1 || report.MergeUnknownSkips[0].Path != wt ||
		!strings.Contains(report.MergeUnknownSkips[0].Reason, "gitea.example") {
		t.Errorf("MergeUnknownSkips = %+v: want %s, naming the forge", report.MergeUnknownSkips, wt)
	}
}

// eitherWayRunner answers "merged" and an error at once, which no runner in
// this package does, but GitRunner is an interface: list must not trust it.
type eitherWayRunner struct{ MockGitRunner }

func (r *eitherWayRunner) IsPRMerged(string, string) (bool, error) {
	return true, errors.New("forge timed out")
}

func TestListNeverReadsAForgeErrorAsMerged(t *testing.T) {
	runner := &eitherWayRunner{MockGitRunner{PorcelainOutput: "worktree /r\nHEAD 1\nbranch refs/heads/main\n\n" +
		"worktree /r-wt\nHEAD 2\nbranch refs/heads/feat/x\n\n"}}
	infos, err := ListWithRunner("/r", runner, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range infos {
		if info.PRMerged {
			t.Errorf("%s read as merged on a forge error", info.Branch)
		}
	}
}
