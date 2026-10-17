package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// laneClaimRepo is a checkout with a lane file whose remote is a local bare
// repository, so the verbs run the real git. The seam is left on the real
// runner: a claim is a property of the push.
func laneClaimRepo(t *testing.T, remote string) string {
	t.Helper()
	top := t.TempDir()
	run := func(dir string, args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run(top, "init", "-q")
	run(top, "remote", "add", "origin", remote)
	if err := os.MkdirAll(filepath.Join(top, ".dotf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(top, ".dotf", "lane.json"), []byte(`{"queue":[2551],"template":".dotf/p.md"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return top
}

func inDir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func TestLaneRelease(t *testing.T) {
	remote := t.TempDir()
	if out, err := exec.Command("git", "-C", remote, "init", "-q", "--bare").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	mine := laneClaimRepo(t, remote)
	other := laneClaimRepo(t, remote)

	inDir(t, other)
	if err := runLane(t, "claim", "2551"); err != nil {
		t.Fatalf("claim from the other checkout: %v", err)
	}

	inDir(t, mine)
	err := runLane(t, "claim", "2551")
	if err == nil || !strings.Contains(err.Error(), "#2551 is claimed by") {
		t.Fatalf("a claim of a held issue = %v, want it refused naming the holder", err)
	}
	if err := runLane(t, "release", "2551"); err == nil {
		t.Fatal("releasing another checkout's claim without --force must be refused")
	}
	if err := runLane(t, "release", "2551", "--force"); err != nil {
		t.Fatalf("--force release: %v", err)
	}
	if err := runLane(t, "release", "2551"); err != nil {
		t.Errorf("releasing a claim nobody holds must be a no-op: %v", err)
	}
	if err := runLane(t, "claim", "2551"); err != nil {
		t.Fatalf("claim after release: %v", err)
	}
	// The same checkout holds it, so its own release needs no --force.
	if err := runLane(t, "release", "2551"); err != nil {
		t.Errorf("release by the holder: %v", err)
	}
	if err := runLane(t, "claim", "not-a-number"); err == nil {
		t.Error("a non-numeric issue must be refused")
	}
}

type claimIssues map[int]lane.Issue

func (m claimIssues) Issue(n int) (lane.Issue, error) {
	if is, ok := m[n]; ok {
		return is, nil
	}
	return lane.Issue{}, errors.New("no such issue")
}

func TestClaimRowsFlagsAClosedIssueAndLeavesAnUnreadableOneOpen(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	at := now.Add(-90 * time.Minute)
	rows := claimRows([]lane.Claim{
		{Issue: 1, Holder: "ana", At: at},
		{Issue: 2, Holder: "bo", At: at},
		{Issue: 3, Holder: "cy"},
	}, claimIssues{1: {State: "CLOSED"}, 2: {State: "OPEN"}}, now)

	if !rows[0].Closed || rows[1].Closed || rows[2].Closed {
		t.Errorf("closed flags = %v %v %v, want true false false (an unreadable issue is never releasable)", rows[0].Closed, rows[1].Closed, rows[2].Closed)
	}
	if rows[0].Age != "1h30m0s" || rows[2].Age != "unknown" {
		t.Errorf("ages = %q and %q", rows[0].Age, rows[2].Age)
	}
	var out strings.Builder
	if err := writeClaimsTable(&out, rows); err != nil || !strings.Contains(out.String(), "dotf lane release 1 --force") {
		t.Errorf("table = %q, %v", out.String(), err)
	}
}

func TestLaneClaimsListsWhatTheRemoteHolds(t *testing.T) {
	remote := t.TempDir()
	if out, err := exec.Command("git", "-C", remote, "init", "-q", "--bare").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	top := laneClaimRepo(t, remote)
	inDir(t, top)
	if err := runLane(t, "claim", "2551"); err != nil {
		t.Fatal(err)
	}
	prev := laneRun
	laneRun = func(name string, args ...string) (string, error) {
		if name == "gh" {
			return `{"number":2551,"state":"CLOSED","title":"t","assignees":[]}`, nil
		}
		return prev(name, args...)
	}
	t.Cleanup(func() { laneRun = prev })

	root := New("test", "test")
	var out strings.Builder
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"lane", "claims"})
	if err := root.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "#2551") || !strings.Contains(out.String(), "issue closed") {
		t.Errorf("output:\n%s", out.String())
	}
}
