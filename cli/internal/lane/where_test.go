package lane

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/worktree"
)

// activeRepo is a repository whose default branch carries the marker and whose
// lanes and activation record a converge run has written.
func activeRepo(t *testing.T) (repo, laneDir string) {
	t.Helper()
	repo, wt := setupLaneTestRepo(t)
	commitMarker(t, repo, true)
	if got, err := Activate(repo, wt, false); err != nil || !got.Active {
		t.Fatalf("Activate = %+v, %v", got, err)
	}
	return repo, filepath.Join(wt, filepath.Base(repo))
}

// holdLane marks the lane's lease ACTIVE, as `dotf lane switch` does.
func holdLane(t *testing.T, lane string) {
	t.Helper()
	meta, err := worktree.LoadMetadata(lane)
	if err != nil || meta == nil {
		t.Fatalf("LoadMetadata(%s) = %v, %v", lane, meta, err)
	}
	meta.State = string(worktree.StateActive)
	if err := worktree.SaveMetadata(lane, *meta); err != nil {
		t.Fatal(err)
	}
}

func TestWhere(t *testing.T) {
	t.Run("a lane of an active repository", func(t *testing.T) {
		_, laneDir := activeRepo(t)
		lane := filepath.Join(laneDir, LaneFeatPrimary)
		runGitLaneTest(t, lane, "switch", "-c", "feat/x")
		holdLane(t, lane)
		got, err := Where(lane)
		if err != nil || got != (Position{Standing: InLane, Lane: LaneFeatPrimary, Branch: "feat/x"}) {
			t.Fatalf("Where(lane) = %+v, %v", got, err)
		}
	})
	t.Run("a lane nobody holds is parked", func(t *testing.T) {
		_, laneDir := activeRepo(t)
		// Detached, as init leaves it.
		got, err := Where(filepath.Join(laneDir, LaneFastTrack))
		if err != nil || got != (Position{Standing: ParkedLane, Lane: LaneFastTrack}) {
			t.Fatalf("Where(detached lane) = %+v, %v", got, err)
		}
		// A branch checked out under a lease still PARKED is not held either.
		lane := filepath.Join(laneDir, LaneFeatSecondary)
		runGitLaneTest(t, lane, "switch", "-c", "feat/y")
		if got, err := Where(lane); err != nil || got.Standing != ParkedLane {
			t.Fatalf("Where(branch under a PARKED lease) = %+v, %v", got, err)
		}
		// And a detached HEAD under an ACTIVE lease is not a place to work.
		holdLane(t, lane)
		runGitLaneTest(t, lane, "switch", "--detach")
		if got, err := Where(lane); err != nil || got.Standing != ParkedLane {
			t.Fatalf("Where(detached under an ACTIVE lease) = %+v, %v", got, err)
		}
	})
	t.Run("the main checkout and a plain worktree are out of a lane", func(t *testing.T) {
		repo, laneDir := activeRepo(t)
		plain := filepath.Join(laneDir, "plain")
		runGitLaneTest(t, repo, "worktree", "add", "-b", "feat/plain", plain)
		for _, dir := range []string{repo, plain, filepath.Join(plain, ".")} {
			got, err := Where(dir)
			if err != nil || got.Standing != OutOfLane {
				t.Errorf("Where(%s) = %+v, %v; want out of a lane", dir, got, err)
			}
		}
	})
	t.Run("no record means inactive, lanes or not", func(t *testing.T) {
		repo, laneDir := activeRepo(t)
		if err := os.Remove(recordPath(t, repo)); err != nil {
			t.Fatal(err)
		}
		for _, dir := range []string{repo, filepath.Join(laneDir, LaneFeatPrimary)} {
			if got, err := Where(dir); err != nil || got.Standing != Inactive {
				t.Errorf("Where(%s) = %+v, %v; want inactive", dir, got, err)
			}
		}
	})
	t.Run("a directory outside any repository is inactive", func(t *testing.T) {
		if got, err := Where(t.TempDir()); err != nil || got.Standing != Inactive {
			t.Fatalf("Where(non-repo) = %+v, %v", got, err)
		}
	})
}

func TestPositionLine(t *testing.T) {
	cases := []struct {
		pos  Position
		want string
	}{
		{Position{Standing: InLane, Lane: "feat-lane-primary", Branch: "feat/x"}, "lane `feat-lane-primary` ACTIVE on `feat/x`"},
		{Position{Standing: ParkedLane, Lane: "fast-track-lane"}, "lane `fast-track-lane` is PARKED: start with `dotf lane switch <branch>`"},
		{Position{Standing: OutOfLane}, "this repository runs lanes: start with `dotf lane switch <branch>`"},
		{Position{Standing: Inactive}, ""},
	}
	for _, c := range cases {
		if got := c.pos.Line(); got != c.want {
			t.Errorf("Line(%+v) = %q, want %q", c.pos, got, c.want)
		}
	}
}
