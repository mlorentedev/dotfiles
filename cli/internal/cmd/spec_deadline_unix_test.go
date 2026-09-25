//go:build !windows

package cmd

import (
	"strings"
	"testing"
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
