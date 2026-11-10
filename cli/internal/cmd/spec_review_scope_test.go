package cmd

import (
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// HARNESS-158: the launch line and --dry-run say how much of the range is the
// spec's own, so the scope is visible before anything runs.
func TestDescribeScopeCountsTheOwnCommits(t *testing.T) {
	scope := spec.ReviewScope{BaseSHA: "abc", Own: spec.OwnCommitSet{RangeSize: 14, Commits: make([]spec.OwnCommit, 5)}}
	if got, want := describeScope(scope), "git diff abc...HEAD (5 of 14 commits are the spec's own)"; got != want {
		t.Errorf("describeScope = %q, want %q", got, want)
	}
	initial := spec.ReviewScope{BaseTree: "tree", RootSHA: "root"}
	if got, want := describeScope(initial), "initial review, git diff tree HEAD (spec first added in root commit root)"; got != want {
		t.Errorf("describeScope(initial) = %q, want %q", got, want)
	}
}
