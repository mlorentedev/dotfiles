package lane

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// removeLaneWorktree takes a lane away the way `gh pr merge --delete-branch`
// did (#3531): git removes the worktree, its lease and its identity with it.
func removeLaneWorktree(t *testing.T, repoDir, wtRoot, name string) string {
	t.Helper()
	lanePath := filepath.Join(wtRoot, "dotfiles", name)
	runGitLaneTest(t, repoDir, "worktree", "remove", "--force", lanePath)
	if _, err := os.Stat(lanePath); err == nil {
		t.Fatalf("%s still exists after git worktree remove", lanePath)
	}
	return lanePath
}

// An absent lane is not an ACTIVE one: counting it as ACTIVE sent the operator
// to `dotf lane park`, which then refused the missing worktree. The fix for an
// absent lane is `dotf lane init`, and the refusal must say so (#3531).
func TestLaneSwitchNamesInitForAnAbsentLane(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)
	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes: %v", err)
	}
	if _, _, err := SwitchLane(repoDir, wtRoot, "chore/1-holds-the-fast-track", ""); err != nil {
		t.Fatalf("SwitchLane: %v", err)
	}
	removeLaneWorktree(t, repoDir, wtRoot, LaneFeatSecondary)

	_, _, err := SwitchLane(repoDir, wtRoot, "chore/2-needs-a-lane", "")
	if err == nil {
		t.Fatal("SwitchLane succeeded with no parked lane and one lane absent")
	}
	msg := err.Error()
	if !strings.Contains(msg, "dotf lane init") || !strings.Contains(msg, LaneFeatSecondary+" is absent") {
		t.Errorf("error %q does not name the absent lane and `dotf lane init`", msg)
	}
	// The lane that is ACTIVE keeps the old advice, and only that lane gets it.
	if !strings.Contains(msg, LaneFastTrack+" is ACTIVE") || !strings.Contains(msg, "dotf lane park") {
		t.Errorf("error %q does not say %s is ACTIVE and can be parked", msg, LaneFastTrack)
	}
	if strings.Contains(msg, "all ACTIVE") || strings.Contains(msg, LaneFeatSecondary+" is ACTIVE") {
		t.Errorf("error %q counts the absent lane as ACTIVE", msg)
	}
}

// A lane whose worktree exists but whose lease is gone is no ACTIVE lane
// either: init writes the lease again.
func TestLaneSwitchNamesInitForALaneWithoutALease(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)
	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes: %v", err)
	}
	if _, _, err := SwitchLane(repoDir, wtRoot, "chore/1-holds-the-fast-track", ""); err != nil {
		t.Fatalf("SwitchLane: %v", err)
	}
	lanePath := filepath.Join(wtRoot, "dotfiles", LaneFeatSecondary)
	gitDir := runGitLaneTest(t, lanePath, "rev-parse", "--absolute-git-dir")
	for _, leaseFile := range []string{filepath.Join(gitDir, "dotf-worktree.json"), filepath.Join(lanePath, ".dotf-worktree.json")} {
		if err := os.Remove(leaseFile); err != nil {
			t.Fatal(err)
		}
	}

	_, _, err := SwitchLane(repoDir, wtRoot, "chore/2-needs-a-lane", "")
	if err == nil || !strings.Contains(err.Error(), LaneFeatSecondary+" has no lease") || !strings.Contains(err.Error(), "dotf lane init") {
		t.Errorf("error %v, want it to name %s without a lease and `dotf lane init`", err, LaneFeatSecondary)
	}
}

// With no lane absent, the refusal is the one it always was.
func TestLaneSwitchWIPCapStillNamesParkWhenEveryLaneIsActive(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)
	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes: %v", err)
	}
	for _, branch := range []string{"chore/1-a", "chore/2-b"} {
		if _, _, err := SwitchLane(repoDir, wtRoot, branch, ""); err != nil {
			t.Fatalf("SwitchLane(%s): %v", branch, err)
		}
	}
	_, _, err := SwitchLane(repoDir, wtRoot, "chore/3-c", "")
	if err == nil || !strings.Contains(err.Error(), "WIP cap") || !strings.Contains(err.Error(), "dotf lane park") || strings.Contains(err.Error(), "dotf lane init") {
		t.Errorf("error %v, want the WIP cap naming `dotf lane park` and not `dotf lane init`", err)
	}
}

func TestLaneParkNamesInitForAnAbsentLane(t *testing.T) {
	repoDir, wtRoot := setupLaneTestRepo(t)
	if _, err := InitLanes(repoDir, wtRoot); err != nil {
		t.Fatalf("InitLanes: %v", err)
	}
	lanePath := removeLaneWorktree(t, repoDir, wtRoot, LaneFeatSecondary)

	_, err := ParkLane(repoDir, lanePath, "", false)
	if err == nil {
		t.Fatal("ParkLane succeeded on a lane that is absent")
	}
	if !strings.Contains(err.Error(), "absent") || !strings.Contains(err.Error(), "dotf lane init") {
		t.Errorf("error %q, want it to say the lane is absent and name `dotf lane init`", err)
	}
	if strings.Contains(err.Error(), "not a persistent named lane") {
		t.Errorf("error %q still calls an absent lane a non-lane", err)
	}
}
