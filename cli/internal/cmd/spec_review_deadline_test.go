package cmd

import (
	"strings"
	"testing"
)

// HARNESS-152: every runner is bounded by the same deadline, in both modes. Only
// agy used to be, through its own --print-timeout; pi ran for as long as it ran.
func TestSpecReviewBoundsTheReviewerInBothModes(t *testing.T) {
	root := makeRepo(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")

	fg, _, err := execute(t, "spec", "review", "AI-001-x", "--foreground", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fg, "'spec' 'deadline' '45m0s'") || !strings.Contains(fg, "'--' '") || !strings.Contains(fg, "'pi' '--print'") {
		t.Errorf("the foreground reviewer is not run under the deadline:\n%s", fg)
	}

	custom, _, err := execute(t, "spec", "review", "AI-001-x", "--foreground", "--dry-run", "--timeout", "10m")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(custom, "'spec' 'deadline' '10m0s'") {
		t.Errorf("--timeout does not reach the deadline:\n%s", custom)
	}

	prev := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/tmux", nil }
	t.Cleanup(func() { lookPath = prev })
	detached, _, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detached, "'tmux' 'new-session'") || !strings.Contains(detached, "deadline") || !strings.Contains(detached, "45m0s") {
		t.Errorf("the detached reviewer is not run under the deadline:\n%s", detached)
	}
}

// HARNESS-152: the prompt tells the reviewer when it will be stopped and when to
// aim to finish, whichever runner reads it.
func TestSpecReviewPromptCarriesTheTimeBudget(t *testing.T) {
	root := makeRepo(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")

	out, _, err := execute(t, "spec", "review", "AI-001-x", "--foreground", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Time budget:") || !strings.Contains(out, "UNVERIFIED") {
		t.Errorf("the reviewer's prompt carries no time budget:\n%s", out)
	}
}

// CLI-094: the stall detector rides on the same wrapper in both modes, watching
// the transcript the launcher announces and reporting against the review file
// the launcher keeps. 10 minutes by default, and --stall-timeout moves it.
func TestSpecReviewWiresTheStallDetectorIntoTheDeadline(t *testing.T) {
	root := makeRepo(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")

	def, _, err := execute(t, "spec", "review", "AI-001-x", "--foreground", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"'--stall-timeout' '10m0s'", "'--transcript'", "review-transcript.jsonl'", "'--review'", "review.md'"} {
		if !strings.Contains(def, want) {
			t.Errorf("the default launch lacks %s:\n%s", want, def)
		}
	}

	custom, _, err := execute(t, "spec", "review", "AI-001-x", "--foreground", "--dry-run", "--stall-timeout", "3m")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(custom, "'--stall-timeout' '3m0s'") {
		t.Errorf("--stall-timeout does not reach the deadline:\n%s", custom)
	}

	if _, _, err := execute(t, "spec", "review", "AI-001-x", "--foreground", "--dry-run", "--stall-timeout", "-1m"); err == nil {
		t.Error("a negative --stall-timeout must be refused")
	}
}
