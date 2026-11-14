package lane

import (
	"os"
	"path/filepath"
	"testing"
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

func TestWhere(t *testing.T) {
	t.Run("a lane of an active repository", func(t *testing.T) {
		_, laneDir := activeRepo(t)
		lane := filepath.Join(laneDir, LaneFeatPrimary)
		runGitLaneTest(t, lane, "switch", "-c", "feat/x")
		got, err := Where(lane)
		if err != nil || got != (Position{Standing: InLane, Lane: LaneFeatPrimary, Branch: "feat/x"}) {
			t.Fatalf("Where(lane) = %+v, %v", got, err)
		}
	})
	t.Run("a parked lane is on no branch", func(t *testing.T) {
		_, laneDir := activeRepo(t)
		got, err := Where(filepath.Join(laneDir, LaneFastTrack))
		if err != nil || got.Standing != InLane || got.Lane != LaneFastTrack || got.Branch != "" {
			t.Fatalf("Where(parked lane) = %+v, %v", got, err)
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
		{Position{Standing: InLane, Lane: "fast-track-lane"}, "lane `fast-track-lane` ACTIVE on `(detached)`"},
		{Position{Standing: OutOfLane}, "this repository runs lanes: start with `dotf lane switch <branch>`"},
		{Position{Standing: Inactive}, ""},
	}
	for _, c := range cases {
		if got := c.pos.Line(); got != c.want {
			t.Errorf("Line(%+v) = %q, want %q", c.pos, got, c.want)
		}
	}
}
