package spec

import (
	"path/filepath"
	"slices"
	"testing"
)

// BUG-166 (#3258): the traced base is the parent of the commit that added the
// spec folder, so a spec committed after its fix hid the fix from the review.
// The base widens to where the branch left the default branch, and only ever
// earlier: a base later than the traced one would hide work, and every case
// below that must not widen pins that it does not.

const widenSpec = "specs/X-166-a"

// forkedFixThenSpec is the #3258 shape: main, then a branch whose fix comes
// before its spec folder. origin/main sits at the fork point, as a fetch leaves
// it. It returns the fork point and the traced base (the commit before the spec).
func forkedFixThenSpec(t *testing.T) (repoRoot, fork, traced string) {
	t.Helper()
	repoRoot = initQuietRepo(t)
	writeAndCommit(t, repoRoot, "base", "README.md")
	fork = trim(git(t, repoRoot, "rev-parse", "HEAD"))
	git(t, repoRoot, "update-ref", "refs/remotes/origin/main", fork)
	git(t, repoRoot, "checkout", "-q", "-b", "fix")
	writeAndCommit(t, repoRoot, "fix(x): stop the leak", "x.go")
	traced = trim(git(t, repoRoot, "rev-parse", "HEAD"))
	writeAndCommit(t, repoRoot, "test: pin the leak, add the spec", widenSpec+"/proposal.md", "x_test.go")
	return repoRoot, fork, traced
}

func TestReviewBaseWidensToTheForkPoint(t *testing.T) {
	repoRoot, fork, traced := forkedFixThenSpec(t)

	got := ResolveReviewBaseDetail(repoRoot, filepath.Join(repoRoot, widenSpec))
	if got.SHA != fork {
		t.Fatalf("base = %s, want the fork point %s (the traced base %s is after the fix)", got.SHA, fork, traced)
	}
	if got.WidenedFrom != traced {
		t.Errorf("WidenedFrom = %q, want the traced base %s", got.WidenedFrom, traced)
	}
	stat := git(t, repoRoot, "diff", "--stat", got.SHA+"...HEAD")
	if !contains(stat, "x.go ") {
		t.Errorf("the review diff must hold the production file; got:\n%s", stat)
	}
}

func TestReviewBaseNeverNarrows(t *testing.T) {
	cases := map[string]func(t *testing.T) (repoRoot, traced string){
		// The spec lands with a squash: on main the fork point is HEAD itself.
		"on main after a squash": func(t *testing.T) (string, string) {
			repoRoot := initQuietRepo(t)
			writeAndCommit(t, repoRoot, "base", "README.md")
			traced := trim(git(t, repoRoot, "rev-parse", "HEAD"))
			writeAndCommit(t, repoRoot, "squash of the spec (#9)", widenSpec+"/proposal.md", "x.go")
			git(t, repoRoot, "update-ref", "refs/remotes/origin/main", "HEAD")
			return repoRoot, traced
		},
		// HARNESS-142: the spec started long ago and main was merged in since,
		// so the fork point is newer than the traced base.
		"main merged into the branch after the spec": func(t *testing.T) (string, string) {
			repoRoot := initQuietRepo(t)
			writeAndCommit(t, repoRoot, "base", "README.md")
			traced := trim(git(t, repoRoot, "rev-parse", "HEAD"))
			git(t, repoRoot, "checkout", "-q", "-b", "work")
			writeAndCommit(t, repoRoot, "slice one", widenSpec+"/proposal.md", "a.go")
			git(t, repoRoot, "checkout", "-q", "main")
			writeAndCommit(t, repoRoot, "other work", "other.go")
			git(t, repoRoot, "update-ref", "refs/remotes/origin/main", "HEAD")
			git(t, repoRoot, "checkout", "-q", "work")
			git(t, repoRoot, "-c", "user.name=t", "-c", "user.email=t@t", "merge", "-q", "--no-ff", "-m", "merge main", "main")
			writeAndCommit(t, repoRoot, "slice two", widenSpec+"/tasks.md", "b.go")
			return repoRoot, traced
		},
		// No remote ref at all: nothing to widen to.
		"no origin ref": func(t *testing.T) (string, string) {
			repoRoot, _, traced := forkedFixThenSpec(t)
			git(t, repoRoot, "update-ref", "-d", "refs/remotes/origin/main")
			return repoRoot, traced
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			repoRoot, traced := build(t)
			got := ResolveReviewBaseDetail(repoRoot, filepath.Join(repoRoot, widenSpec))
			if got.SHA != traced || got.WidenedFrom != "" {
				t.Fatalf("base = %s (widened from %q), want the traced base %s unchanged", got.SHA, got.WidenedFrom, traced)
			}
		})
	}
}

// The fix that the widening exists to include names no issue and does not
// touch the spec folder; listed or not, it must not read as other work.
func TestOwnCommitsCountTheWorkBeforeTheSpec(t *testing.T) {
	repoRoot, _, _ := forkedFixThenSpec(t)
	base := ResolveReviewBaseDetail(repoRoot, filepath.Join(repoRoot, widenSpec))

	set, err := OwnCommitsWidened(repoRoot, base.SHA, base.WidenedFrom, "HEAD",
		SpecFolderNames(repoRoot, filepath.Join(repoRoot, widenSpec)), 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"fix(x): stop the leak", "test: pin the leak, add the spec"}
	if got := subjects(set); !slices.Equal(got, want) || set.RangeSize != 2 {
		t.Fatalf("own = %v of %d, want %v of 2", got, set.RangeSize, want)
	}
}
