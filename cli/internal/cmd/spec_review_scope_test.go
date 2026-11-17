package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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
	ownCommitsOf = func(string, string, string, string, string, int) (spec.OwnCommitSet, error) {
		return spec.OwnCommitSet{}, errors.New("git broke")
	}
	_, _, err := reviewRange(t.TempDir(), t.TempDir(), "X-001-a", false)
	if err == nil || !strings.Contains(err.Error(), "git broke") {
		t.Errorf("reviewRange err = %v, want the listing failure", err)
	}
}

// issueRepo is a repository whose origin is GitHub and whose spec names issue.
func issueRepo(t *testing.T, issue string) (repoRoot, specDir string) {
	t.Helper()
	t.Setenv("DOTF_BITACORA_REPO", "")
	repoRoot = t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", "https://github.com/mlorentedev/dotfiles.git"}} {
		if out, err := exec.Command("git", append([]string{"-C", repoRoot}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	specDir = filepath.Join(repoRoot, "specs", "X-001-a")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: X-001-a\n"
	if issue != "" {
		body += "issue: \"" + issue + "\"\n"
	}
	if err := os.WriteFile(filepath.Join(specDir, "proposal.md"), []byte(body+"---\n# p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return repoRoot, specDir
}

// The issue rule fires only through specIssueNumber, so drive reviewRange with
// a real proposal.md and origin and check the issue reaches the listing.
func TestReviewRangeMatchesTheSpecsOwnIssue(t *testing.T) {
	repoRoot, specDir := issueRepo(t, "mlorentedev/dotfiles#1727")
	prevBase, prevHead, prevOwn := resolveReviewBase, headSHAOf, ownCommitsOf
	t.Cleanup(func() { resolveReviewBase, headSHAOf, ownCommitsOf = prevBase, prevHead, prevOwn })
	resolveReviewBase = func(string, string) spec.ReviewBase { return spec.ReviewBase{SHA: "basebasebase"} }
	headSHAOf = func(string) string { return "headheadhead" }
	var got int
	ownCommitsOf = func(_, _, _, _, _ string, issue int) (spec.OwnCommitSet, error) {
		got = issue
		return spec.OwnCommitSet{RangeSize: 2, Commits: make([]spec.OwnCommit, 1)}, nil
	}
	scope, _, err := reviewRange(repoRoot, specDir, "X-001-a", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1727 || scope.OwnIssue != 1727 || scope.OwnFolderOnly != "" {
		t.Errorf("issue passed = %d, scope = %+v; want 1727 and no folder-only reason", got, scope)
	}
}

func TestSpecIssueNumberFallsBackToTheFolderAndSaysWhy(t *testing.T) {
	for _, tc := range []struct{ issue, why string }{
		{"mlorentedev/kubelab#5", "lives outside mlorentedev/dotfiles"},
		{"", "names no issue"},
		{"not a ref", "does not parse"},
		{"#0", "not a positive integer"},
	} {
		repoRoot, specDir := issueRepo(t, tc.issue)
		n, why := specIssueNumber(repoRoot, specDir)
		if n != 0 || !strings.Contains(why, tc.why) {
			t.Errorf("issue %q: got (%d, %q), want 0 and a reason containing %q", tc.issue, n, why, tc.why)
		}
	}
	scope := spec.ReviewScope{BaseSHA: "abc", OwnFolderOnly: "proposal.md names no issue", Own: spec.OwnCommitSet{RangeSize: 3, Commits: make([]spec.OwnCommit, 1)}}
	if got, want := describeScope(scope), "git diff abc...HEAD (1 of 3 commits are the spec's own; folder rule only: proposal.md names no issue)"; got != want {
		t.Errorf("describeScope = %q, want %q", got, want)
	}
}

// BUG-166 (#3258): a base widened to the fork point says so on the launch line,
// and the widened range reaches the own-commit listing with its traced base.
func TestReviewScopeSaysTheBaseWasWidened(t *testing.T) {
	prevBase, prevHead, prevOwn := resolveReviewBase, headSHAOf, ownCommitsOf
	t.Cleanup(func() { resolveReviewBase, headSHAOf, ownCommitsOf = prevBase, prevHead, prevOwn })
	resolveReviewBase = func(string, string) spec.ReviewBase {
		return spec.ReviewBase{SHA: "forkforkfork", WidenedFrom: "tracedtraced0000"}
	}
	headSHAOf = func(string) string { return "headheadhead" }
	var gotWidened string
	ownCommitsOf = func(_, _, _, widenedFrom, _ string, _ int) (spec.OwnCommitSet, error) {
		gotWidened = widenedFrom
		return spec.OwnCommitSet{RangeSize: 2, Commits: make([]spec.OwnCommit, 2)}, nil
	}
	scope, _, err := reviewRange(t.TempDir(), t.TempDir(), "X-001-a", false)
	if err != nil {
		t.Fatal(err)
	}
	if gotWidened != "tracedtraced0000" {
		t.Errorf("own commits listed with widenedFrom %q, want the traced base", gotWidened)
	}
	want := "git diff forkforkfork...HEAD (2 of 2 commits are the spec's own; folder rule only: proposal.md is unreadable; base widened from tracedtraced to where the branch left the default branch)"
	if got := describeScope(scope); got != want {
		t.Errorf("describeScope = %q, want %q", got, want)
	}
	var b strings.Builder
	b.WriteString(spec.ReviewPromptForScope(spec.FirstSigner, "X-001-a", "/r", "rev", "run", "skill", scope))
	if !strings.Contains(b.String(), "tracedtraced") {
		t.Errorf("the prompt must name the traced base the scope was widened from:\n%s", b.String())
	}
}
