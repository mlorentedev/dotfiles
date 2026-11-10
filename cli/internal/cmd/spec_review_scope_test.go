package cmd

import (
	"errors"
	"strings"
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

// A git failure while listing the own commits refuses the launch: an empty list
// would read as a spec with no commits of its own.
func TestReviewRangeRefusesWhenTheOwnCommitsCannotBeListed(t *testing.T) {
	prevBase, prevHead, prevOwn := resolveReviewBase, headSHAOf, ownCommitsOf
	t.Cleanup(func() { resolveReviewBase, headSHAOf, ownCommitsOf = prevBase, prevHead, prevOwn })
	resolveReviewBase = func(string, string) spec.ReviewBase { return spec.ReviewBase{SHA: "basebasebase"} }
	headSHAOf = func(string) string { return "headheadhead" }
	ownCommitsOf = func(string, string, string, string, int) (spec.OwnCommitSet, error) {
		return spec.OwnCommitSet{}, errors.New("git broke")
	}
	_, _, err := reviewRange(t.TempDir(), t.TempDir(), "X-001-a", false)
	if err == nil || !strings.Contains(err.Error(), "git broke") {
		t.Errorf("reviewRange err = %v, want the listing failure", err)
	}
}
