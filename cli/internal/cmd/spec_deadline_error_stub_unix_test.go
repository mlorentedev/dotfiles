//go:build !windows

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BUG-168 AC1 (#1511): a reviewer that ends without writing its verdict leaves
// a review file with verdict ERROR in the caller's spec folder, saying how the
// run ended and what the reviewer wrote, and stderr says it was written.
func TestDeadlineWritesAnErrorStubWhenTheRunSignedNothing(t *testing.T) {
	for _, c := range []struct {
		name, limit, script, ended string
		code                       int
	}{
		{"exit 0, nothing written", "1m", `echo "Individual quota reached." >&2`, "exited 0 without writing review.md", 0},
		{"runner failed", "1m", `echo "Individual quota reached." >&2; exit 3`, "exited with status 3", 3},
		{"deadline", "2s", `echo "Individual quota reached." >&2; sleep 30`, "deadline", 124},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, callerSpec, tree := reviewWorktree(t)
			review := filepath.Join(tree, "specs", "AI-001-x", "review.md")
			req := `{"reviewed_sha":"abc123","reviewer":"agy/gemini-3.1-pro-high"}`
			if err := os.WriteFile(filepath.Join(filepath.Dir(review), "review-request.json"), []byte(req), 0o644); err != nil {
				t.Fatal(err)
			}
			stderrFile := filepath.Join(callerSpec, "review-transcript.jsonl.stderr")
			script := "{ " + c.script + "; } 2> " + stderrFile

			_, stderr, err := execute(t, "spec", "deadline", c.limit, "--review", review,
				"--worktree", tree, "--copy-back", callerSpec, "--", "sh", "-c", script)
			if got := ExitCode(err); got != c.code {
				t.Fatalf("exit %d, want %d, the run's own (err %v)\n%s", got, c.code, err, stderr)
			}
			raw, rerr := os.ReadFile(filepath.Join(callerSpec, "review.md"))
			if rerr != nil {
				t.Fatalf("no review.md reached the caller's spec folder: %v\n%s", rerr, stderr)
			}
			for _, want := range []string{"verdict: ERROR", "agy/gemini-3.1-pro-high", "abc123", c.ended, "Individual quota reached."} {
				if !strings.Contains(string(raw), want) {
					t.Errorf("review.md must contain %q:\n%s", want, raw)
				}
			}
			if !strings.Contains(stderr, "verdict ERROR") {
				t.Errorf("stderr does not say an ERROR record was written:\n%s", stderr)
			}
		})
	}
}

// BUG-168 AC2: the reviewer's own file is copied back as it was written.
func TestDeadlineKeepsTheReviewersOwnFile(t *testing.T) {
	_, callerSpec, tree := reviewWorktree(t)
	review := filepath.Join(tree, "specs", "AI-001-x", "review.md")
	_, stderr, _ := execute(t, "spec", "deadline", "1m", "--review", review,
		"--worktree", tree, "--copy-back", callerSpec, "--",
		"sh", "-c", `printf -- '---\nverdict: PENDING\n---\n' > "$1"; exit 3`, "sh", review)
	raw, err := os.ReadFile(filepath.Join(callerSpec, "review.md"))
	if err != nil || !strings.Contains(string(raw), "PENDING") || strings.Contains(string(raw), "ERROR") {
		t.Errorf("review.md = %q (err %v), want the reviewer's PENDING file untouched\n%s", raw, err, stderr)
	}
}
