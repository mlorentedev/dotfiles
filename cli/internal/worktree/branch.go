package worktree

import (
	"fmt"
	"regexp"
	"strings"
)

// branchShape is <type>/<kebab-slug>: a lowercase Conventional Commit type, one
// "/", then lowercase alphanumeric words joined by single hyphens.
var branchShape = regexp.MustCompile(`^[a-z]+/[a-z0-9]+(-[a-z0-9]+)*$`)

// PlanningTokens are the words harness/enforced/no-phase-references.md bans
// from branch names. That file is extracted from the vault's git-workflow
// pattern; widen the pattern first, then this list.
// TestPlanningTokensMatchTheEnforcedRule fails when they disagree.
var PlanningTokens = []string{"phase", "milestone"}

// planningWord matches a planning token as a whole hyphen-separated word,
// optionally followed by digits: phase-3 and milestone2 but not phaseout.
var planningWord = regexp.MustCompile(`(?:^|-)(` + strings.Join(PlanningTokens, "|") + `)[0-9]*(?:-|$)`)

// ValidateBranch refuses a branch add should not create, naming the rule it
// breaks. The shape rule makes herdr's slug of the branch the branch itself
// with "/" as "-", so either name finds the worktree.
//
// It gates creation only. migrate, adopt and done act on branches that already
// exist, and so does add when the branch is already there (a release or
// dependabot branch): none of them refuses a name it did not choose.
func ValidateBranch(branch string) error {
	if !branchShape.MatchString(branch) {
		return fmt.Errorf("branch %q is not <type>/<kebab-slug> (lowercase type, \"/\", lowercase words joined by \"-\")", branch)
	}
	_, slug, _ := strings.Cut(branch, "/")
	if m := planningWord.FindStringSubmatch(slug); m != nil {
		return fmt.Errorf("branch %q carries the planning token %q: phases and milestones belong on the board, not in branch names (no-phase-references); name the change instead, e.g. feat/scaffold-pyhydra3d", branch, m[1])
	}
	return nil
}
