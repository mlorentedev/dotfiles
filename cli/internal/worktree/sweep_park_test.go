package worktree

import (
	"errors"
	"os"
	"path/filepath"
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

		if !parkSingleNamedLane(repo, info, &RealSweepRunner{}, heads) {
			t.Fatal("did not park a lane whose tip is the head of a merged pull request")
		}
		if got := laneState(t, info.Path); got != "PARKED" {
			t.Errorf("state = %q, want PARKED", got)
		}
		if out := runGitParkTest(t, info.Path, "status", "--porcelain=v2", "--branch"); !slices.Contains(strings.Split(out, "\n"), "# branch.head (detached)") {
			t.Errorf("lane still on a branch after parking:\n%s", out)
		}
	})

	t.Run("a lane whose landing cannot be checked is not parked", func(t *testing.T) {
		repo, info, _ := parkLaneFixture(t)
		heads := func(_, _ string) ([]string, error) { return nil, errors.New("forge unreachable") }

		if parkSingleNamedLane(repo, info, &RealSweepRunner{}, heads) {
			t.Fatal("parked a lane although nothing could say it landed")
		}
		if got := laneState(t, info.Path); got != "ACTIVE" {
			t.Errorf("state = %q, want ACTIVE", got)
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

		if parkSingleNamedLane(repo, info, &RealSweepRunner{}, heads) {
			t.Fatal("reported a lane as parked although its checkout could not detach")
		}
		if got := laneState(t, info.Path); got == "PARKED" {
			t.Error("state left PARKED on a lane that is still on its branch")
		}
		if got := runGitParkTest(t, info.Path, "symbolic-ref", "--short", "HEAD"); got != "feat/lane-a" {
			t.Errorf("HEAD = %q, want the lane's branch", got)
		}
	})
}
