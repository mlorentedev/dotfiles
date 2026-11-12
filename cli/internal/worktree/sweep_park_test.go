package worktree

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// parkLaneFixture builds a repository with a pushed main and a named lane on
// its own branch with one commit, the shape sweep parks once the lane's work
// has landed. It returns the repository, the lane's Info and the lane's tip.
func parkLaneFixture(t *testing.T) (repoDir string, info Info, tip string) {
	t.Helper()
	root := t.TempDir()
	originDir := filepath.Join(root, "origin.git")
	repoDir = filepath.Join(root, "repo")
	wtDir := filepath.Join(root, "lanes", "lane-a")
	for _, d := range []string{originDir, repoDir, filepath.Dir(wtDir)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	runGitParkTest(t, originDir, "init", "--bare", "-b", "main")
	runGitParkTest(t, repoDir, "init", "-b", "main")
	runGitParkTest(t, repoDir, "remote", "add", "origin", originDir)
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitParkTest(t, repoDir, "add", "README.md")
	runGitParkTest(t, repoDir, "commit", "-m", "initial")
	runGitParkTest(t, repoDir, "push", "-u", "origin", "main")
	runGitParkTest(t, repoDir, "worktree", "add", "--no-track", "-b", "feat/lane-a", wtDir, "origin/main")
	ensureExclude(repoDir, MetadataFileName)
	if err := os.WriteFile(filepath.Join(wtDir, "feature.txt"), []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitParkTest(t, wtDir, "add", "feature.txt")
	runGitParkTest(t, wtDir, "commit", "-m", "feat work")
	tip = runGitParkTest(t, wtDir, "rev-parse", "HEAD")

	meta := Metadata{
		Lane:           "lane-a",
		State:          "ACTIVE",
		CreatedAt:      time.Now().Add(-2 * time.Hour),
		LeaseExpiresAt: time.Now().Add(-time.Hour),
	}
	if err := SaveMetadata(wtDir, meta); err != nil {
		t.Fatal(err)
	}
	return repoDir, Info{Path: wtDir, Branch: "feat/lane-a", Metadata: &meta}, tip
}

func laneState(t *testing.T, path string) string {
	t.Helper()
	m, err := LoadMetadata(path)
	if err != nil || m == nil {
		t.Fatalf("LoadMetadata(%s): %v", path, err)
	}
	return m.State
}

// The outcomes #3263 asks for, through sweep's own park path.
func TestParkSingleNamedLane_Outcomes(t *testing.T) {
	t.Run("a squash-merged lane parks when the landed heads name its tip", func(t *testing.T) {
		repo, info, tip := parkLaneFixture(t)
		heads := func(_, _ string) ([]string, error) { return []string{tip}, nil }

		if parked, why := parkSingleNamedLane(repo, info, &RealSweepRunner{}, heads); !parked {
			t.Fatalf("did not park a lane whose tip is the head of a merged pull request: %s", why)
		}
		if got := laneState(t, info.Path); got != "PARKED" {
			t.Errorf("state = %q, want PARKED", got)
		}
		if out := runGitParkTest(t, info.Path, "status", "--porcelain=v2", "--branch"); !slices.Contains(strings.Split(out, "\n"), "# branch.head (detached)") {
			t.Errorf("lane still on a branch after parking:\n%s", out)
		}
	})

	t.Run("a lane with no landing reader is not parked", func(t *testing.T) {
		repo, info, _ := parkLaneFixture(t)
		mergedPRHeads = nil
		t.Cleanup(func() { mergedPRHeads = ghMergedPRHeads })

		// The mock answers "not merged" without asking the forge; the real
		// runner would reach the nil reader first (list.go) and panic.
		parked, why := parkSingleNamedLane(repo, info, &MockSweepRunner{}, nil)
		if parked {
			t.Fatal("parked a lane whose landing no reader could decide")
		}
		if why != "its landing could not be decided" {
			t.Errorf("why = %q, want the undecided landing", why)
		}
		if got := laneState(t, info.Path); got != "ACTIVE" {
			t.Errorf("state = %q, want ACTIVE", got)
		}
	})
	t.Run("a lane whose lease cannot be written is not parked", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("a read-only directory does not refuse a write here")
		}
		repo, info, tip := parkLaneFixture(t)
		heads := func(_, _ string) ([]string, error) { return []string{tip}, nil }
		// The lease is written in place, so the file itself refuses.
		lease := filepath.Join(info.Path, MetadataFileName)
		if err := os.Chmod(lease, 0o444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(lease, 0o644) })

		parked, why := parkSingleNamedLane(repo, info, &RealSweepRunner{}, heads)
		if parked {
			t.Fatal("parked a lane whose lease could not be written")
		}
		if !strings.HasPrefix(why, "its lease could not be written: ") || !strings.Contains(why, "permission denied") {
			t.Errorf("why = %q, want the lease write's error", why)
		}
		if got := runGitParkTest(t, info.Path, "symbolic-ref", "--short", "HEAD"); got != "feat/lane-a" {
			t.Errorf("HEAD = %q, want the lane's branch", got)
		}
	})
	t.Run("a lane whose landing cannot be checked is not parked", func(t *testing.T) {
		repo, info, _ := parkLaneFixture(t)
		heads := func(_, _ string) ([]string, error) { return nil, errors.New("forge unreachable") }

		parked, why := parkSingleNamedLane(repo, info, &RealSweepRunner{}, heads)
		if parked {
			t.Fatal("parked a lane although nothing could say it landed")
		}
		if !strings.Contains(why, "could not check for a merged pull request") || !strings.Contains(why, "forge unreachable") {
			t.Errorf("reason %q does not say the landing could not be checked, nor why", why)
		}
		if got := laneState(t, info.Path); got != "ACTIVE" {
			t.Errorf("state = %q, want ACTIVE", got)
		}
	})

	// Every conflicting tree #3263 names shows in status (an untracked file or a
	// modification), so the lane is refused before any state is written.
	t.Run("a lane with an untracked file is not parked", func(t *testing.T) {
		repo, info, tip := parkLaneFixture(t)
		heads := func(_, _ string) ([]string, error) { return []string{tip}, nil }
		if err := os.WriteFile(filepath.Join(info.Path, "README.md.orig"), []byte("scratch\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		parked, why := parkSingleNamedLane(repo, info, &RealSweepRunner{}, heads)
		if parked {
			t.Fatal("parked a lane holding an untracked file")
		}
		if !strings.Contains(why, "uncommitted") {
			t.Errorf("reason %q does not name the dirty tree", why)
		}
		if got := laneState(t, info.Path); got != "ACTIVE" {
			t.Errorf("state = %q, want ACTIVE", got)
		}
		if got := runGitParkTest(t, info.Path, "symbolic-ref", "--short", "HEAD"); got != "feat/lane-a" {
			t.Errorf("HEAD = %q, want the lane's branch", got)
		}
	})

	t.Run("a lane git refuses to detach keeps its state", func(t *testing.T) {
		repo, info, tip := parkLaneFixture(t)
		heads := func(_, _ string) ([]string, error) { return []string{tip}, nil }
		// A held index lock makes checkout fail while status still reads clean.
		gitDir := runGitParkTest(t, info.Path, "rev-parse", "--absolute-git-dir")
		if err := os.WriteFile(filepath.Join(gitDir, "index.lock"), nil, 0o644); err != nil {
			t.Fatal(err)
		}

		parked, why := parkSingleNamedLane(repo, info, &RealSweepRunner{}, heads)
		if parked {
			t.Fatal("reported a lane as parked although its checkout could not detach")
		}
		if !strings.Contains(why, "detach") || !strings.Contains(why, "index.lock") {
			t.Errorf("reason %q does not name the refused detach and git's error", why)
		}
		if got := laneState(t, info.Path); got == "PARKED" {
			t.Error("state left PARKED on a lane that is still on its branch")
		}
		if got := runGitParkTest(t, info.Path, "symbolic-ref", "--short", "HEAD"); got != "feat/lane-a" {
			t.Errorf("HEAD = %q, want the lane's branch", got)
		}
	})
}

// The part of #3263 that #3310 left open: a lane sweep did not park was one
// more in the skipped count, and the reason parkSingleNamedLane had already
// computed was dropped. The report names the lane and the reason.
func TestSweepReportsALaneItDidNotPark(t *testing.T) {
	withHostProcessInside(t, false)
	repo, info, tip := parkLaneFixture(t)
	// Landed on main, so sweep classifies the lane as parkable; the held
	// index lock then makes the detach fail while status still reads clean.
	runGitParkTest(t, repo, "merge", "--ff-only", "feat/lane-a")
	runGitParkTest(t, repo, "push", "origin", "main")
	gitDir := runGitParkTest(t, info.Path, "rev-parse", "--absolute-git-dir")
	if err := os.WriteFile(filepath.Join(gitDir, "index.lock"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := SweepWithRunner(SweepOptions{
		RepoRoot:    repo,
		LockPath:    filepath.Join(t.TempDir(), "sweep.lock"),
		LandedHeads: func(_, _ string) ([]string, error) { return []string{tip}, nil },
	}, &RealSweepRunner{}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Reaped) != 0 {
		t.Fatalf("reaped %v: want nothing", report.Reaped)
	}
	// Two skips: the main checkout, never a candidate, and the lane.
	if report.SkippedCount != 2 {
		t.Errorf("SkippedCount = %d, want 2 (the main checkout and the lane)", report.SkippedCount)
	}
	if len(report.LaneSkips) != 1 || !samePath(report.LaneSkips[0].Path, info.Path) ||
		!strings.Contains(report.LaneSkips[0].Reason, "detach") || !strings.Contains(report.LaneSkips[0].Reason, "index.lock") {
		t.Errorf("LaneSkips = %+v: want %s with the refused detach as the reason", report.LaneSkips, info.Path)
	}
	if got := laneState(t, info.Path); got != "ACTIVE" {
		t.Errorf("state = %q, want ACTIVE", got)
	}
}

// The JSON name is the wire contract the report's readers hold on to.
func TestSweepReportNamesLaneSkipsOnTheWire(t *testing.T) {
	out, err := json.Marshal(SweepReport{LaneSkips: []SessionSkip{{Path: "p", Reason: "r"}}})
	if err != nil {
		t.Fatal(err)
	}
	if want := `"lane_skips":[{"path":"p","reason":"r"}]`; !strings.Contains(string(out), want) {
		t.Errorf("report = %s, want %s", out, want)
	}
	if out, _ = json.Marshal(SweepReport{}); strings.Contains(string(out), "lane_skips") {
		t.Errorf("empty report still carries lane_skips: %s", out)
	}
}
