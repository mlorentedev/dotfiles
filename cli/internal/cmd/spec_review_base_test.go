package cmd

import (
	"errors"
	"strings"
	"testing"
)

// HARNESS-112. A review whose diff is empty does not fail — it silently becomes
// a reading of the spec folder, and reports findings with nothing executed
// behind them. That is worse than no review, because it arrives wearing a
// verdict. The launcher must refuse instead.

func withReviewBase(t *testing.T, base, head string) {
	t.Helper()
	prevBase, prevHead := resolveReviewBase, headSHAOf
	resolveReviewBase = func(string, string) string { return base }
	headSHAOf = func(string) string { return head }
	t.Cleanup(func() { resolveReviewBase, headSHAOf = prevBase, prevHead })
}

func seedReviewFixture(t *testing.T) string {
	t.Helper()
	root := makeRepo(t)
	writeFixture(t, root, "harness/reviewer-pool.json", `{"pool":[
		{"id":"nan/deepseek-v4-flash","runner":"pi","provider":"nan","model":"deepseek-v4-flash","role":"primary"}
	]}`)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	prev := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("no tmux") }
	t.Cleanup(func() { lookPath = prev })
	return root
}

func TestSpecReviewRefusesWhenNoBaseCanBeResolved(t *testing.T) {
	seedReviewFixture(t)
	withReviewBase(t, "", "headheadhead")

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err == nil {
		t.Fatalf("the launch was allowed with no review base; the reviewer would browse the "+
			"spec folder and call that a review:\n%s", stdout+stderr)
	}
	if !strings.Contains(err.Error(), "cannot resolve a review base") {
		t.Errorf("the refusal does not say what is wrong: %v", err)
	}
}

func TestSpecReviewRefusesWhenTheBaseIsTheHead(t *testing.T) {
	seedReviewFixture(t)
	withReviewBase(t, "samesamesame", "samesamesame")

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err == nil {
		t.Fatalf("a review with base == HEAD was allowed; its diff is empty by construction:\n%s",
			stdout+stderr)
	}
	if !strings.Contains(err.Error(), "nothing to review") {
		t.Errorf("the refusal does not name the empty diff: %v", err)
	}
}

// The positive direction, so the two refusals above cannot pass vacuously by the
// launcher refusing everything.
func TestSpecReviewLaunchesAndStatesTheBaseWhenOneResolves(t *testing.T) {
	seedReviewFixture(t)
	withReviewBase(t, "basebasebase", "headheadhead")

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err != nil {
		t.Fatalf("a resolvable base was refused: %v\n%s", err, stdout+stderr)
	}
	if out := stdout + stderr; !strings.Contains(out, "basebasebase") {
		t.Errorf("the launch does not hand the reviewer its base, so it will guess `main` — "+
			"which is empty on a post-merge review:\n%s", out)
	}
}
