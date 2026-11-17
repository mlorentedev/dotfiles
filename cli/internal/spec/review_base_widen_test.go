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

// Local main ahead of a stale origin/main, the spec in the later commit: the
// base widens to the stale fork, which is wider and hides nothing.
func TestReviewBaseWidensPastAStaleOriginRef(t *testing.T) {
	repoRoot := initQuietRepo(t)
	writeAndCommit(t, repoRoot, "base", "README.md")
	stale := trim(git(t, repoRoot, "rev-parse", "HEAD"))
	git(t, repoRoot, "update-ref", "refs/remotes/origin/main", stale)
	writeAndCommit(t, repoRoot, "landed, not fetched", "other.go")
	traced := trim(git(t, repoRoot, "rev-parse", "HEAD"))
	writeAndCommit(t, repoRoot, "squash of the spec (#9)", widenSpec+"/proposal.md", "x.go")

	got := ResolveReviewBaseDetail(repoRoot, filepath.Join(repoRoot, widenSpec))
	if got.SHA != stale || got.WidenedFrom != traced {
		t.Fatalf("base = %s widened from %q, want %s widened from %s", got.SHA, got.WidenedFrom, stale, traced)
	}
}

// The BUG-108 rename trace runs first; the widening applies to the base it
// traced, not to the parent of the rename.
func TestReviewBaseWidensAfterTracingARename(t *testing.T) {
	repoRoot := initQuietRepo(t)
	writeAndCommit(t, repoRoot, "base", "README.md")
	fork := trim(git(t, repoRoot, "rev-parse", "HEAD"))
	git(t, repoRoot, "update-ref", "refs/remotes/origin/main", fork)
	git(t, repoRoot, "checkout", "-q", "-b", "fix")
	writeAndCommit(t, repoRoot, "fix(x): stop the leak", "x.go")
	traced := trim(git(t, repoRoot, "rev-parse", "HEAD"))
	writeAndCommit(t, repoRoot, "add the spec", "specs/X-166-old/proposal.md")
	git(t, repoRoot, "mv", "specs/X-166-old", widenSpec)
	git(t, repoRoot, "commit", "-qm", "rename the spec")

	specDir := filepath.Join(repoRoot, widenSpec)
	got := ResolveReviewBaseDetail(repoRoot, specDir)
	if got.SHA != fork || got.WidenedFrom != traced {
		t.Fatalf("base = %s widened from %q, want %s widened from %s", got.SHA, got.WidenedFrom, fork, traced)
	}
	if names := SpecFolderNames(repoRoot, specDir); !slices.Equal(names, []string{widenSpec, "specs/X-166-old"}) {
		t.Errorf("folder names = %v, want both names traced", names)
	}
}

// The prompt's own-commit rule names the work before the spec folder, so the
// listed fix does not contradict the rule printed above it.
func TestReviewPromptNamesTheWidenedOwnRule(t *testing.T) {
	scope := ReviewScope{BaseSHA: "fork", WidenedFrom: "traced",
		Own: OwnCommitSet{RangeSize: 1, Commits: []OwnCommit{{SHA: "abc", Subject: "fix(x): stop the leak"}}}}
	prompt := ReviewPromptForScope(FirstSigner, "X-166-a", "/r", "rev", "run", "skill", scope)
	if !contains(prompt, "or come before the spec folder on this branch") {
		t.Errorf("the own-commit rule must name the widened rule:\n%s", prompt)
	}
}
