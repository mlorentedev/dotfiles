package prland

import (
	"context"
	"strings"
	"testing"
)

// A refusal that a human may override carries the exact command that merges the
// head Land read, so the manual path is the same on every refusal (#3108). The
// command pins that head: gh refuses it if the head has moved.
func TestLand_ARefusalCarriesTheManualMergeCommand(t *testing.T) {
	redReview := `[{"name":"test","bucket":"pass"},{"name":"review","bucket":"fail"}]`
	for name, tc := range map[string]struct {
		repo string
		want string
	}{
		"current repository": {"", "gh pr merge 30 --squash --match-head-commit abc123"},
		"named repository":   {"o/r", "gh pr merge 30 --squash --match-head-commit abc123 --repo o/r"},
	} {
		t.Run(name, func(t *testing.T) {
			g := &fakeGH{view: readyView, checks: redReview, deps: `[]`}

			res, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged, Repo: tc.repo}, 30)
			if err != nil {
				t.Fatal(err)
			}
			if res.Merged || len(g.calls) != 0 {
				t.Fatalf("want a refusal and no mutation, got %+v, calls %v", res, g.calls)
			}
			if res.ManualMerge != tc.want {
				t.Errorf("ManualMerge = %q, want %q", res.ManualMerge, tc.want)
			}
		})
	}
}

// Where the refusal is not about a check, or a PR gh would merge anyway, a
// command would hand the reader a bypass: a PR that is not open or is a draft
// cannot be merged, and the grant's own refusals (a release, an excluded
// branch, a change to the grant, no opt-in) are the owner's, not land's to
// route around.
func TestLand_ARefusalThatIsNotACheckCarriesNoManualCommand(t *testing.T) {
	cases := map[string]*fakeGH{
		"draft":           {view: strings.Replace(readyView, `"isDraft":false`, `"isDraft":true`, 1)},
		"closed":          {view: strings.Replace(readyView, `"state":"OPEN"`, `"state":"CLOSED"`, 1)},
		"release-please":  {view: strings.Replace(readyView, `"headRefName":"feat/x"`, `"headRefName":"release-please--branches--main"`, 1)},
		"excluded branch": {view: strings.Replace(readyView, `"headRefName":"feat/x"`, `"headRefName":"deploy/x"`, 1), grantYAML: grantFixture + "  - branch:deploy/\n"},
		"changes grant":   {view: readyView, files: `{"filename":"` + GrantFile + `","previous_filename":null}`},
		"no grant":        {view: readyView, noGrant: true},
	}
	for name, g := range cases {
		t.Run(name, func(t *testing.T) {
			g.checks, g.deps = greenChecks, `[]`

			res, err := Land(context.Background(), Options{Run: g.run, Untriaged: noneUntriaged}, 30)
			if err != nil {
				t.Fatal(err)
			}
			if res.Merged || len(res.Reasons) == 0 {
				t.Fatalf("want a refusal, got %+v", res)
			}
			if res.ManualMerge != "" {
				t.Errorf("ManualMerge = %q, want none", res.ManualMerge)
			}
		})
	}
}

// Reviewer output awaiting triage is the gate the standing grant rests on: land
// does not print the way around it.
func TestLand_AnUntriagedPRCarriesNoManualCommand(t *testing.T) {
	g := &fakeGH{view: readyView, checks: greenChecks, deps: `[]`}
	untriaged := func(context.Context) ([]int, error) { return []int{30}, nil }

	res, _ := Land(context.Background(), Options{Run: g.run, Untriaged: untriaged}, 30)

	if res.Merged || res.ManualMerge != "" {
		t.Errorf("want a refusal with no manual command, got %+v", res)
	}
}
