package lane

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
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
	runGitLaneTest(t, repo, "-c", "core.hooksPath="+t.TempDir(), "commit", "-m", "declare lanes")
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

// The plan predicts the apply for a lane that exists but is half provisioned:
// a plan that said 0 where the apply writes 1 would break converge's contract.
func TestActivate_PlanPredictsApplyForAHalfProvisionedLane(t *testing.T) {
	damage := map[string]func(t *testing.T, lanePath string){
		"identity record missing": func(t *testing.T, lanePath string) {
			gitDir, _, err := worktreeGitDirs(lanePath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(gitDir, LaneIdentityFile)); err != nil {
				t.Fatal(err)
			}
		},
		"lease only in the tree (legacy)": func(t *testing.T, lanePath string) {
			gitDir, _, err := worktreeGitDirs(lanePath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(gitDir, worktree.LeaseFileName)); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(lanePath, worktree.MetadataFileName), mustJSON(t, worktree.Metadata{Lane: LaneFastTrack, State: "PARKED"}), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"no lease anywhere": func(t *testing.T, lanePath string) {
			gitDir, _, err := worktreeGitDirs(lanePath)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range []string{filepath.Join(gitDir, worktree.LeaseFileName), filepath.Join(lanePath, worktree.MetadataFileName)} {
				if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
			}
		},
	}
	for name, hurt := range damage {
		t.Run(name, func(t *testing.T) {
			repo, wt := setupLaneTestRepo(t)
			commitMarker(t, repo, true)
			if _, err := Activate(repo, wt, false); err != nil {
				t.Fatal(err)
			}
			hurt(t, filepath.Join(wt, filepath.Base(repo), LaneFastTrack))

			plan, err := Activate(repo, wt, true)
			if err != nil || plan.Changes != 1 {
				t.Fatalf("plan = %+v, %v; want 1 change", plan, err)
			}
			applied, err := Activate(repo, wt, false)
			if err != nil || applied.Changes != plan.Changes {
				t.Fatalf("apply = %+v, %v; want what the plan said (%d)", applied, err, plan.Changes)
			}
			if again, err := Activate(repo, wt, true); err != nil || again.Changes != 0 {
				t.Fatalf("re-plan = %+v, %v; want 0", again, err)
			}
		})
	}
}

// A lane path another repository (or nothing git knows) occupies is refused,
// as a RefusedError, in the plan and in the apply alike, before any lane or
// record is written.
func TestActivate_RefusesAnOccupiedLanePathBeforeWritingAnything(t *testing.T) {
	repo, wt := setupLaneTestRepo(t)
	commitMarker(t, repo, true)
	laneDir := filepath.Join(wt, filepath.Base(repo))
	if err := os.MkdirAll(filepath.Join(laneDir, LaneFastTrack), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dry := range []bool{true, false} {
		_, err := Activate(repo, wt, dry)
		var refused *RefusedError
		if !errors.As(err, &refused) {
			t.Fatalf("dryRun=%v: err = %v; want a RefusedError", dry, err)
		}
	}
	if _, err := os.Stat(recordPath(t, repo)); !os.IsNotExist(err) {
		t.Errorf("a record was written despite the refusal (err=%v)", err)
	}
	if _, err := os.Stat(filepath.Join(laneDir, LaneFeatPrimary)); !os.IsNotExist(err) {
		t.Errorf("another lane was created despite the refusal (err=%v)", err)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
