package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// A park names what it archived and what it kept (MEMORY-020 AC1, AC3).
func TestReportParkThreadsNamesArchivedAndKept(t *testing.T) {
	var out bytes.Buffer
	a := mem.ParkArchive{Path: "/v/archive/superseded-handoffs-2026-11-14.md", Archived: []string{"feat-a"}, Kept: []string{"feat-a+agy"}}
	if err := reportParkThreads(&out, "/wt/lane", a, nil); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`archived   thread "feat-a" to /v/archive/superseded-handoffs-2026-11-14.md`, `kept       thread "feat-a+agy" in MEMORY.md: it still has Awaiting Manu items`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
}

// A failed archive becomes the command's error, in ERROR / WHY / FIX form,
// and says the lane is parked (MEMORY-020 AC4).
func TestReportParkThreadsTurnsAFailedArchiveIntoTheError(t *testing.T) {
	err := reportParkThreads(&bytes.Buffer{}, "/wt/lane", mem.ParkArchive{}, errors.New("disk full"))
	if err == nil {
		t.Fatal("a failed archive did not fail the command")
	}
	for _, want := range []string{"ERROR: lane /wt/lane is parked", "WHY: disk full", "FIX: ", "--finished"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}

// A sweep reports every lane whose archive failed and fails on them.
func TestReportSweepThreadsFailsOnAnArchiveFailure(t *testing.T) {
	var out bytes.Buffer
	r := &worktree.SweepReport{
		ArchivedThreads: []string{"feat-a"},
		ArchiveFailures: []worktree.SessionSkip{{Path: "/wt/one", Reason: "disk full"}, {Path: "/wt/two", Reason: "locked"}},
	}
	err := reportSweepThreads(&out, r)
	if err == nil || !strings.Contains(err.Error(), "lane /wt/one is parked") {
		t.Fatalf("err %v, want the first failure", err)
	}
	if !strings.Contains(out.String(), "lane /wt/two is parked") || !strings.Contains(out.String(), "archived thread: feat-a") {
		t.Errorf("output lacks the second failure or the archived thread:\n%s", out.String())
	}
	if err := reportSweepThreads(&out, &worktree.SweepReport{}); err != nil {
		t.Errorf("a clean sweep failed: %v", err)
	}
}
