package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const gateReviewComment = `[{"user":{"login":"github-actions[bot]"},"created_at":"2026-09-25T10:00:00Z","body":"## PR Reviewer Guide"}]`

func gateCommits(n int) string {
	one := `{"sha":"c","parents":[{"sha":"p"}],"commit":{"author":{"date":"2026-09-25T11:00:00Z"}}}`
	return "[" + strings.TrimSuffix(strings.Repeat(one+",", n), ",") + "]"
}

// gateFiles writes the two inputs and returns the flags that name them.
func gateFiles(t *testing.T, comments, commits string) []string {
	t.Helper()
	dir := t.TempDir()
	c, m := filepath.Join(dir, "comments.json"), filepath.Join(dir, "commits.json")
	for path, content := range map[string]string{c: comments, m: commits} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return []string{"--comments", c, "--commits", m}
}

func TestPrGate_PrintsItsDecision(t *testing.T) {
	stdout, stderr, err := execute(t, append([]string{"pr", "gate"}, gateFiles(t, gateReviewComment, gateCommits(3))...)...)
	if err != nil {
		t.Fatalf("pr gate: %v\n%s", err, stderr)
	}
	want := "run=true\nreason=3 new commit(s) since the review of 2026-09-25T10:00:00Z, at or past the threshold of 3\nmode=incremental\n"
	if stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

func TestPrGate_ThresholdAndSenderReachTheDecision(t *testing.T) {
	flags := gateFiles(t, gateReviewComment, gateCommits(1))
	stdout, _, err := execute(t, append(append([]string{"pr", "gate"}, flags...), "--threshold", "1")...)
	if err != nil || !strings.HasPrefix(stdout, "run=true\n") {
		t.Errorf("--threshold 1: stdout = %q, err = %v", stdout, err)
	}
	stdout, _, err = execute(t, append(append([]string{"pr", "gate"}, flags...), "--sender-type", "Bot")...)
	if err != nil || !strings.HasPrefix(stdout, "run=false\n") {
		t.Errorf("--sender-type Bot: stdout = %q, err = %v", stdout, err)
	}
}

// Fails open: a file it cannot read is a decision to review, never an error.
func TestPrGate_AnAbsentFileReviews(t *testing.T) {
	dir := t.TempDir()
	stdout, stderr, err := execute(t, "pr", "gate", "--comments", filepath.Join(dir, "absent.json"),
		"--commits", filepath.Join(dir, "absent2.json"))
	if err != nil {
		t.Fatalf("pr gate: %v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "run=true") || !strings.Contains(stdout, "mode=full") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestPrGate_AUsageErrorExitsTwo(t *testing.T) {
	files := gateFiles(t, gateReviewComment, gateCommits(1))
	for name, args := range map[string][]string{
		"no flags":             {},
		"no --commits":         files[:2],
		"an empty --comments":  {"--comments", "", "--commits", files[3]},
		"an argument":          append([]string{"extra"}, files...),
		"an unknown flag":      append([]string{"--nope"}, files...),
		"a flag with no value": append(append([]string{}, files...), "--threshold"),
		"a word as threshold":  append(append([]string{}, files...), "--threshold", "many"),
		"an empty threshold":   append(append([]string{}, files...), "--threshold", ""),
		"a negative threshold": append(append([]string{}, files...), "--threshold", "-1"),
		"a signed threshold":   append(append([]string{}, files...), "--threshold", "+3"),
		// The script's `[ -ge ]` errors on this and falls through to run=false.
		"a threshold past int": append(append([]string{}, files...), "--threshold", "99999999999999999999999"),
	} {
		t.Run(name, func(t *testing.T) {
			stdout, _, err := execute(t, append([]string{"pr", "gate"}, args...)...)
			if ExitCode(err) != 2 {
				t.Errorf("exit code = %d (%v), want 2", ExitCode(err), err)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want no decision", stdout)
			}
		})
	}
}

func TestPrGate_HelpNamesTheThreshold(t *testing.T) {
	stdout, _, err := execute(t, "pr", "gate", "--help")
	if err != nil || !strings.Contains(stdout, "--threshold") {
		t.Errorf("stdout = %q, err = %v", stdout, err)
	}
}
