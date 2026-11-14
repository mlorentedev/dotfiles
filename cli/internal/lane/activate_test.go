package lane

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// commitMarker writes the marker through its writer (lesson 569) and commits
// it to the local main, and to origin's main when push is set.
func commitMarker(t *testing.T, repo string, push bool) {
	t.Helper()
	if _, err := WriteLanesMarker(repo); err != nil {
		t.Fatal(err)
	}
	runGitLaneTest(t, repo, "add", LanesMarkerFile)
	// The machine's own guard must not decide whether a fixture can commit.
	runGitLaneTest(t, repo, "-c", "core.hooksPath="+os.DevNull, "commit", "-m", "declare lanes")
	if push {
		runGitLaneTest(t, repo, "push", "origin", "main")
	}
}

func recordPath(t *testing.T, repo string) string {
	t.Helper()
	return filepath.Join(runGitLaneTest(t, repo, "rev-parse", "--path-format=absolute", "--git-common-dir"), ActivationFile)
}

func assertInactive(t *testing.T, repo, wt string) {
	t.Helper()
	got, err := Activate(repo, wt, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active || got.Changes != 0 {
		t.Errorf("Activate = %+v, want inactive with no change", got)
	}
	if _, err := os.Stat(recordPath(t, repo)); !os.IsNotExist(err) {
		t.Errorf("an activation record exists (err=%v)", err)
	}
	if _, err := os.Stat(filepath.Join(wt, filepath.Base(repo), LaneFeatPrimary)); !os.IsNotExist(err) {
		t.Errorf("a lane was created (err=%v)", err)
	}
}

// The marker counts only where it is committed on the default branch: a copy
// in the working tree, or on a local commit nobody pushed, activates nothing.
func TestActivate_ReadsTheCommittedDefaultBranchNotTheWorkingTree(t *testing.T) {
	t.Run("working tree only", func(t *testing.T) {
		repo, wt := setupLaneTestRepo(t)
		if _, err := WriteLanesMarker(repo); err != nil {
			t.Fatal(err)
		}
		assertInactive(t, repo, wt)
	})
	t.Run("committed locally, not on the remote default branch", func(t *testing.T) {
		repo, wt := setupLaneTestRepo(t)
		commitMarker(t, repo, false)
		assertInactive(t, repo, wt)
	})
	t.Run("no marker anywhere", func(t *testing.T) {
		repo, wt := setupLaneTestRepo(t)
		assertInactive(t, repo, wt)
	})
}

func TestActivate_CreatesTheLanesAndTheRecordOnceThenReportsNothing(t *testing.T) {
	repo, wt := setupLaneTestRepo(t)
	commitMarker(t, repo, true)

	plan, err := Activate(repo, wt, true)
	if err != nil || !plan.Active || plan.Changes != len(CanonicalLanes)+1 {
		t.Fatalf("plan = %+v, %v; want active with %d changes", plan, err, len(CanonicalLanes)+1)
	}
	if _, err := os.Stat(recordPath(t, repo)); !os.IsNotExist(err) {
		t.Fatalf("a plan wrote the record (err=%v)", err)
	}

	got, err := Activate(repo, wt, false)
	if err != nil || !got.Active || got.Changes != plan.Changes {
		t.Fatalf("apply = %+v, %v; want what the plan said (%d)", got, err, plan.Changes)
	}
	for _, name := range CanonicalLanes {
		if _, err := os.Stat(filepath.Join(wt, filepath.Base(repo), name)); err != nil {
			t.Errorf("lane %s: %v", name, err)
		}
	}
	first, err := os.ReadFile(recordPath(t, repo))
	if err != nil {
		t.Fatalf("no activation record: %v", err)
	}

	again, err := Activate(repo, wt, false)
	if err != nil || !again.Active || again.Changes != 0 {
		t.Fatalf("second run = %+v, %v; want active and changes=0", again, err)
	}
	if second, _ := os.ReadFile(recordPath(t, repo)); !bytes.Equal(first, second) {
		t.Errorf("the record changed on a re-run: %q then %q", first, second)
	}
}

// The record is not in any working tree, so a clean of every worktree leaves it.
func TestActivate_RecordSurvivesGitClean(t *testing.T) {
	repo, wt := setupLaneTestRepo(t)
	commitMarker(t, repo, true)
	if _, err := Activate(repo, wt, false); err != nil {
		t.Fatal(err)
	}
	runGitLaneTest(t, repo, "clean", "-fdx")
	if _, err := os.Stat(recordPath(t, repo)); err != nil {
		t.Errorf("git clean -fdx removed the record: %v", err)
	}
}

// A linked worktree names the same repository: the lanes land under the main
// checkout's name, not the worktree's.
func TestActivate_FromALinkedWorktreeUsesTheMainCheckout(t *testing.T) {
	repo, wt := setupLaneTestRepo(t)
	commitMarker(t, repo, true)
	linked := filepath.Join(t.TempDir(), "linked")
	runGitLaneTest(t, repo, "worktree", "add", "--detach", linked)

	if got, err := Activate(linked, wt, false); err != nil || got.Changes != len(CanonicalLanes)+1 {
		t.Fatalf("Activate = %+v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(wt, "dotfiles", LaneFeatPrimary)); err != nil {
		t.Errorf("lanes were not created under the main checkout's name: %v", err)
	}
}

// A hand-run init provisions lanes and nothing else: it neither writes the
// marker nor the activation record, whatever marker the repository carries.
func TestInitLanes_NeverWritesTheRecordOrTheMarker(t *testing.T) {
	cases := map[string]func(t *testing.T, repo string){
		"no marker":            func(*testing.T, string) {},
		"uncommitted marker":   func(t *testing.T, repo string) { _, _ = WriteLanesMarker(repo) },
		"marker on the remote": func(t *testing.T, repo string) { commitMarker(t, repo, true) },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			repo, wt := setupLaneTestRepo(t)
			setup(t, repo)
			_, hadMarker := os.Stat(filepath.Join(repo, filepath.FromSlash(LanesMarkerFile)))
			if _, err := InitLanes(repo, wt); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(recordPath(t, repo)); !os.IsNotExist(err) {
				t.Errorf("init wrote the activation record (err=%v)", err)
			}
			if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(LanesMarkerFile))); (err == nil) != (hadMarker == nil) {
				t.Errorf("init changed whether %s exists (before err=%v, after err=%v)", LanesMarkerFile, hadMarker, err)
			}
		})
	}
}
