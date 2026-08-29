package cmd

import (
	"errors"
	"strings"
	"testing"
)

// The existing spec-review tests were written when the default was the pool's
// first entry; pinning the draw to index 0 keeps them meaning what they meant.
func init() { reviewerDraw = func(int) int { return 0 } }

// HARNESS-093 (#1370): with no --reviewer the launcher draws a member and says so;
// the draw can reach any index, not only the first.
func TestSpecReviewDrawsAPoolMemberAndSaysSo(t *testing.T) {
	orig := reviewerDraw
	t.Cleanup(func() { reviewerDraw = orig })
	reviewerDraw = func(n int) int { return n - 1 }
	root := makeRepo(t)
	writeFixture(t, root, "harness/reviewer-pool.json", `{"pool":[
		{"id":"nan/deepseek-v4-flash","runner":"pi","provider":"nan","model":"deepseek-v4-flash","role":"primary"},
		{"id":"nan/glm5.3-flash","runner":"pi","provider":"nan","model":"glm5.3-flash","role":"member"}
	]}`)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	prev := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("no tmux") }
	t.Cleanup(func() { lookPath = prev })

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err != nil {
		t.Fatalf("%v\n%s", err, stdout+stderr)
	}
	if out := stdout + stderr; !strings.Contains(out, "Reviewer:   nan/glm5.3-flash (pi, random draw)") {
		t.Errorf("the launch line must name the drawn member and say it was drawn:\n%s", out)
	}
	stdout, stderr, err = execute(t, "spec", "review", "AI-001-x", "--dry-run", "--reviewer", "nan/deepseek-v4-flash")
	if err != nil {
		t.Fatalf("%v\n%s", err, stdout+stderr)
	}
	if out := stdout + stderr; !strings.Contains(out, "Reviewer:   nan/deepseek-v4-flash (pi, requested)") {
		t.Errorf("--reviewer must be reported as requested:\n%s", out)
	}
}
