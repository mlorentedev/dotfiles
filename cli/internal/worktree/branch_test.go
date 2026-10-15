package worktree

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestValidateBranch(t *testing.T) {
	for _, tc := range []struct {
		branch string
		rule   string // "" when the branch is accepted
	}{
		{"feat/foo-bar", ""},
		{"fix/bug-143", ""},
		{"feat/worktree-one-root", ""},
		{"foo", "<type>/<kebab-slug>"},
		{"Feat/x", "<type>/<kebab-slug>"},
		{"feat/Foo", "<type>/<kebab-slug>"},
		{"feat/a/b", "<type>/<kebab-slug>"},
		{"feat/foo--bar", "<type>/<kebab-slug>"},
		{"feat/foo_bar", "<type>/<kebab-slug>"},
		{"feat/phase-3-x", "no-phase-references"},
		{"feat/milestone2", "no-phase-references"},
		{"feat/phase3", "no-phase-references"},
		{"feat/x-milestone-2", "no-phase-references"},
		{"chore/cleanup-phase", "no-phase-references"},
		// Near misses stay accepted: a plain prefix match would refuse them.
		{"feat/phaseout-legacy", ""},
		{"fix/milestones-view", ""},
		{"fix/emphase", ""},
	} {
		err := ValidateBranch(tc.branch)
		switch {
		case tc.rule == "" && err != nil:
			t.Errorf("%s: refused: %v", tc.branch, err)
		case tc.rule != "" && (err == nil || !strings.Contains(err.Error(), tc.rule)):
			t.Errorf("%s: got %v, want a refusal naming %q", tc.branch, err, tc.rule)
		}
	}
}

// The shape rule exists so herdr's slug of a branch is the branch with "/" as
// "-": every branch add accepts must keep that.
func TestAcceptedBranchesSlugToThemselves(t *testing.T) {
	for _, b := range []string{"feat/foo-bar", "fix/bug-143", "a/b1-c2", "feat/phaseout-legacy"} {
		if err := ValidateBranch(b); err != nil {
			t.Fatalf("%s: %v", b, err)
		}
		if got, want := slugifyBranch(b), strings.ReplaceAll(b, "/", "-"); got != want {
			t.Errorf("slug of %s = %s, want %s", b, got, want)
		}
	}
}

// PlanningTokens must be the words of the enforced rule, so the validator and
// the instruction every agent reads cannot drift apart.
func TestPlanningTokensMatchTheEnforcedRule(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "..", "harness", "enforced", "no-phase-references.md"))
	if err != nil {
		t.Fatal(err)
	}
	// The rule names its words in its heading, "No internal phase/milestone
	// references". Both directions: a word the list adds and one the rule adds
	// each fail, and a reworded heading fails rather than passing on nothing.
	m := regexp.MustCompile(`\*\*No internal ([a-z/]+) references\*\*`).FindSubmatch(doc)
	if m == nil {
		t.Fatal("harness/enforced/no-phase-references.md no longer names its words as **No internal <a>/<b> references**; update this test")
	}
	banned := strings.Split(string(m[1]), "/")
	slices.Sort(banned)
	got := slices.Sorted(slices.Values(PlanningTokens))
	if !slices.Equal(got, banned) {
		t.Errorf("PlanningTokens = %q, the enforced rule bans %q", got, banned)
	}
}

// add validates the branch it will create, given or defaulted, before it
// touches the disk.
func TestAddRefusesAnInvalidBranchBeforeCreatingAnything(t *testing.T) {
	for _, opts := range []AddOptions{
		{Slug: "x", Branch: "feat/phase-2-x"},
		{Slug: "Foo_Bar"}, // defaults to feat/Foo_Bar
		{Slug: "x", Branch: "Feat/x", CustomPath: filepath.Join(t.TempDir(), "custom")},
	} {
		opts.RepoRoot = filepath.Join(t.TempDir(), "myrepo")
		opts.WorktreeRoot = filepath.Join(t.TempDir(), "worktrees")
		_ = os.MkdirAll(filepath.Join(opts.RepoRoot, ".git", "info"), 0o755)
		called := false
		runner := &MockAddRunner{OnWorktreeAdd: func(_, _, _, _ string) error { called = true; return nil }}

		if _, err := AddWithRunner(opts, runner, time.Now()); err == nil || !strings.Contains(err.Error(), "branch") {
			t.Errorf("%+v: got %v, want a branch refusal", opts, err)
		}
		if called {
			t.Errorf("%+v: git worktree add ran for an invalid branch", opts)
		}
	}
}
