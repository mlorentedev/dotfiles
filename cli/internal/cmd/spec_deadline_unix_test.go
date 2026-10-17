//go:build !windows

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// HARNESS-152: a run the deadline stops exits 124 and says the time limit ended
// it, so nobody reads a missing review.md as a crash or as a verdict.
func TestSpecDeadlineStopsTheRunAndSaysSo(t *testing.T) {
	_, stderr, err := execute(t, "spec", "deadline", "1s", "--", "sh", "-c", "sleep 30")
	if got := ExitCode(err); got != 124 {
		t.Fatalf("exit %d, want 124 (err %v)", got, err)
	}
	if !strings.Contains(stderr, "deadline") || !strings.Contains(stderr, "not a verdict") {
		t.Errorf("stderr does not say the deadline ended the run:\n%s", stderr)
	}
}

// A run that ends in time keeps its own exit status.
func TestSpecDeadlinePassesTheRunnersStatusThrough(t *testing.T) {
	_, _, err := execute(t, "spec", "deadline", "10s", "--", "sh", "-c", "exit 3")
	if got := ExitCode(err); got != 3 {
		t.Errorf("exit %d, want the runner's 3 (err %v)", got, err)
	}
}

// CLI-094 (#1975): a transcript that stops changing ends the run at the stall
// timeout, not the cap. The status is its own (not 124), and the stop names the
// last tool call and says what review.md holds, so the next reader has the
// cause and the state without opening the transcript.
func TestSpecDeadlineStopsAStalledRunAndNamesTheLastCall(t *testing.T) {
	dir := t.TempDir()
	transcript := filepath.Join(dir, "review-transcript.jsonl")
	if err := os.WriteFile(transcript, []byte(`{"type":"tool_execution_update","toolName":"bash","args":{"command":"find ~/Projects"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(transcript, old, old); err != nil {
		t.Fatal(err)
	}
	review := filepath.Join(dir, "review.md")

	_, stderr, err := execute(t, "spec", "deadline", "1m", "--stall-timeout", "400ms",
		"--transcript", transcript, "--review", review, "--", "sh", "-c", "sleep 30")
	if got := ExitCode(err); got != 125 {
		t.Fatalf("exit %d, want 125, the stall's own status (err %v)", got, err)
	}
	for _, want := range []string{"stalled", "bash: find ~/Projects", "review.md was not rewritten by this run", "not a verdict"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr must mention %q:\n%s", want, stderr)
		}
	}
}

// CLI-094 AC4: the cap's stop reports the same fact, here a provisional record
// the reviewer wrote first, and still exits non-zero.
func TestSpecDeadlineReportsAProvisionalReviewKeptAtTheCap(t *testing.T) {
	dir := t.TempDir()
	review := filepath.Join(dir, "review.md")
	if err := os.WriteFile(review, []byte("---\nverdict: PENDING\nreviewer: pi\n---\n\nfindings so far\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := execute(t, "spec", "deadline", "1s", "--review", review, "--", "sh", "-c", "sleep 30")
	if got := ExitCode(err); got != 124 {
		t.Fatalf("exit %d, want 124 (err %v)", got, err)
	}
	if !strings.Contains(stderr, "provisional review.md kept (verdict PENDING)") {
		t.Errorf("stderr does not say the provisional review was kept:\n%s", stderr)
	}
}
