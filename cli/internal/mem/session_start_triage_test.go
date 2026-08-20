package mem

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The triage section exists to close the half of the review loop GUARD-002 leaves
// open. These tests pin the three states that half can be in, and the one that
// matters most is the middle one: a queue that could not be computed must never
// render like a queue with nothing in it.

func TestTriageQueueSection(t *testing.T) {
	t.Run("pending PRs are named", func(t *testing.T) {
		got := triageQueue("#1085, #1088", nil)
		if !strings.Contains(got, "#1085, #1088") {
			t.Fatalf("want the PR refs in the section, got %q", got)
		}
		if !strings.Contains(got, "[pr-triage]") {
			t.Fatalf("want the section tag, got %q", got)
		}
		if !strings.HasPrefix(got, "\n") {
			t.Fatalf("every brief section opens with a newline, got %q", got)
		}
	})

	t.Run("an empty queue is silent", func(t *testing.T) {
		if got := triageQueue("", nil); got != "" {
			t.Fatalf("an empty queue must add nothing to the brief, got %q", got)
		}
		if got := triageQueue("   \n ", nil); got != "" {
			t.Fatalf("whitespace is an empty queue, got %q", got)
		}
	})

	// The load-bearing one. `dotf pr triage-queue` exits non-zero when it cannot
	// answer, precisely so an unanswerable queue is not mistaken for an empty one.
	// If this section swallowed the error, the brief would restore exactly the
	// blind spot the exit contract was written to close.
	t.Run("a failure is reported, never rendered as empty", func(t *testing.T) {
		got := triageQueue("", errors.New("gh pr list: exit status 4"))
		if got == "" {
			t.Fatal("a queue that could not be computed must not be silent")
		}
		if !strings.Contains(got, "exit status 4") {
			t.Fatalf("want the underlying reason surfaced, got %q", got)
		}
		if !strings.Contains(got, "not an empty queue") {
			t.Fatalf("want the message to deny the empty reading outright, got %q", got)
		}
	})

	t.Run("a failure wins over a stale summary", func(t *testing.T) {
		got := triageQueue("#42", errors.New("boom"))
		if strings.Contains(got, "#42") {
			t.Fatalf("a summary alongside an error must not be reported as fact, got %q", got)
		}
	})
}

// A nil probe is how a caller says "this repository does not run the loop". It
// must be indistinguishable from the section not existing — not an empty queue,
// and certainly not an error.
func TestBriefSkipsTriageWhenProbeIsNil(t *testing.T) {
	dir := t.TempDir()
	opts := BriefOptions{Cwd: dir, StaleDays: 14, Now: time.Now()}

	if got := Brief(opts); strings.Contains(got, "[pr-triage]") {
		t.Fatalf("a nil probe must add no section at all, got %q", got)
	}

	opts.TriageQueue = func() (string, error) { return "#7", nil }
	if got := Brief(opts); !strings.Contains(got, "[pr-triage]") {
		t.Fatalf("a wired probe must reach the agnostic brief, got %q", got)
	}

	opts.TriageQueue = func() (string, error) { return "", errors.New("timeout querying queue") }
	got := Brief(opts)
	if !strings.Contains(got, "[pr-triage]") || !strings.Contains(got, "timeout querying queue") || !strings.Contains(got, "not an empty queue") {
		t.Fatalf("an error probe must render loud failure in the agnostic brief, got %q", got)
	}
}

// The agnostic brief is what opencode, agy and copilot consume. Wiring the probe
// into the Claude adapter alone would close the loop in one harness and leave it
// open in the others, which is the asymmetry this CLI exists to remove.
func TestClaudeContextCarriesTriageSection(t *testing.T) {
	dir := t.TempDir()
	in := ClaudeContextInput{
		Cwd:         dir,
		Now:         time.Now(),
		TriageQueue: func() (string, error) { return "#1085", nil },
	}
	// Without a .git dir the git-gated block is skipped, so the section is absent.
	if got := ClaudeContext(in); strings.Contains(got, "[pr-triage]") {
		t.Fatalf("outside a git repo there is no PR loop to close, got %q", got)
	}

	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ClaudeContext(in); !strings.Contains(got, "#1085") {
		t.Fatalf("inside a git repo the section must render, got %q", got)
	}

	in.TriageQueue = func() (string, error) { return "", errors.New("gh api unreachable") }
	got := ClaudeContext(in)
	if !strings.Contains(got, "[pr-triage]") || !strings.Contains(got, "gh api unreachable") || !strings.Contains(got, "not an empty queue") {
		t.Fatalf("an error probe must render loud failure in Claude context, got %q", got)
	}
}
