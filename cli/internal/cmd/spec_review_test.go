package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// seedPool writes a repo whose pool holds one pi-backed reviewer.
func seedPool(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "harness")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	pool := `{"pool":[{"id":"nan/deepseek-v4-flash","runner":"pi","provider":"nan","model":"deepseek-v4-flash","role":"primary"}]}`
	if err := os.WriteFile(filepath.Join(dir, "reviewer-pool.json"), []byte(pool), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The foreground path must NOT go through a shell, for two reasons that a
// pipeline cannot satisfy:
//
//  1. In POSIX sh the status of `cmd | tee f` is tee's, and tee almost always
//     succeeds — so a reviewer that crashed or timed out would report success.
//     dash has no `set -o pipefail` to fix that.
//  2. `sh` is not on PATH on stock Windows, and this path is precisely the
//     fallback the command documents for machines without tmux.
//
// The fix runs the reviewer directly with io.MultiWriter, so this test pins the
// absence of a shell rather than the presence of one.
func TestSpecReviewForegroundDoesNotShellOut(t *testing.T) {
	root := makeRepo(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--foreground", "--dry-run")
	if err != nil {
		t.Fatalf("spec review --foreground --dry-run: %v\n%s", err, stdout+stderr)
	}
	out := stdout + stderr
	for _, forbidden := range []string{"'sh' '-c'", "| tee "} {
		if strings.Contains(out, forbidden) {
			t.Errorf("the foreground path must not build a shell pipeline (found %q):\n%s", forbidden, out)
		}
	}
	if !strings.Contains(out, "'pi' '--print'") {
		t.Errorf("the reviewer must be invoked directly:\n%s", out)
	}
}

// The dry-run line has to be runnable as printed. The tmux form's last element
// is an entire pipeline, so joining the raw elements with spaces yields a line
// that a human pasting it would hand to tmux as several arguments instead of one.
//
// tmux presence is stubbed rather than detected: otherwise this test asserts
// one thing on a developer machine and another in a container that lacks tmux,
// which is how a test comes to pass everywhere and prove nothing.
func TestSpecReviewDryRunPrintsAQuotedCommand(t *testing.T) {
	root := makeRepo(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")

	prev := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/tmux", nil }
	t.Cleanup(func() { lookPath = prev })

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err != nil {
		t.Fatalf("spec review --dry-run: %v\n%s", err, stdout+stderr)
	}
	out := stdout + stderr

	// Each tmux argument quoted individually...
	if !strings.Contains(out, "'tmux' 'new-session' '-d'") {
		t.Errorf("tmux arguments must be quoted for display:\n%s", out)
	}
	// ...and the pipeline, being ONE argument, quoted as a whole — which is
	// exactly what a raw strings.Join would have lost.
	if !strings.Contains(out, `'\''dotf'\''`) {
		t.Errorf("the pipeline element must be quoted as a single argument:\n%s", out)
	}
}

// A model outside the pool is refused before a reviewer is ever spawned —
// cheaper than the archive gate catching it afterwards, and it names what IS
// available rather than only what is forbidden.
func TestSpecReviewRefusesAnUnpooledReviewer(t *testing.T) {
	root := makeRepo(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")

	_, _, err := execute(t, "spec", "review", "AI-001-x", "--reviewer", "claude-opus-5", "--dry-run")
	if err == nil {
		t.Fatal("a model outside the pool must not be launchable")
	}
	if !strings.Contains(err.Error(), "nan/deepseek-v4-flash") {
		t.Errorf("the refusal must list the available reviewers, got: %v", err)
	}
}

// --- BUG-081: the launcher must not announce a review it never started -------
//
// `tmux new-session -d` returns 0 once the SESSION exists, which says nothing
// about the process inside it. Observed on HARNESS-072: the reviewer died on a
// broken credential, the session was gone within half a second, and the
// launcher printed "[OK] Review running detached" over a 0-byte transcript.

// stubLaunch makes the detached path runnable in a test: tmux "exists", the
// new-session call "succeeds", and the sleep between liveness probes is free.
// alive decides whether the session outlives the probe window.
func stubLaunch(t *testing.T, alive bool) {
	t.Helper()
	prevLook, prevRun, prevAlive, prevSleep := lookPath, runCommand, sessionAlive, sleepFor
	lookPath = func(string) (string, error) { return "/usr/bin/tmux", nil }
	runCommand = func(string, []string) error { return nil }
	sessionAlive = func(string) bool { return alive }
	sleepFor = func(time.Duration) {}
	t.Cleanup(func() {
		lookPath, runCommand, sessionAlive, sleepFor = prevLook, prevRun, prevAlive, prevSleep
	})
}

func TestSpecReviewFailsWhenTheLaunchDiedImmediately(t *testing.T) {
	root := makeRepo(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	stubLaunch(t, false)

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x")
	out := stdout + stderr
	if err == nil {
		t.Fatalf("a launch whose session is already gone must fail:\n%s", out)
	}
	if strings.Contains(out, "Review running detached") {
		t.Errorf("announced a running review over a dead one:\n%s", out)
	}
}

// The error has to carry what the reviewer said, not just that it died. The
// death reason arrives on stderr, which the `| tee` pipeline never captured —
// that is why the original failure left no clue anywhere.
func TestSpecReviewQuotesTheReviewerStderrWhenItDies(t *testing.T) {
	root := makeRepo(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	stubLaunch(t, false)

	stderrFile := filepath.Join(root, "specs", "AI-001-x", "review-transcript.jsonl.stderr")
	if err := os.WriteFile(stderrFile, []byte("Error: bw resolve dockerhub/password: not found\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := execute(t, "spec", "review", "AI-001-x")
	if err == nil {
		t.Fatal("expected the dead launch to fail")
	}
	if !strings.Contains(err.Error(), "bw resolve dockerhub/password") {
		t.Errorf("the error must quote what the reviewer wrote, got: %v", err)
	}
}

func TestSpecReviewSuggestsAlternativePoolMemberOnDeath(t *testing.T) {
	root := makeRepo(t)
	dir := filepath.Join(root, "harness")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	pool := `{"pool":[
		{"id":"nan/deepseek-v4-flash","runner":"pi","provider":"nan","model":"deepseek-v4-flash","role":"primary"},
		{"id":"nan/mimo-v2.5","runner":"pi","provider":"nan","model":"mimo-v2.5","role":"fallback"}
	]}`
	if err := os.WriteFile(filepath.Join(dir, "reviewer-pool.json"), []byte(pool), 0o644); err != nil {
		t.Fatal(err)
	}
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	stubLaunch(t, false)

	_, _, err := execute(t, "spec", "review", "AI-001-x")
	if err == nil {
		t.Fatal("expected the dead launch to fail")
	}
	if !strings.Contains(err.Error(), "dotf spec review AI-001-x --reviewer nan/mimo-v2.5") {
		t.Errorf("expected error to suggest the next pool member, got: %v", err)
	}
}

func TestSpecReviewAnnouncesALaunchThatSurvived(t *testing.T) {
	root := makeRepo(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	stubLaunch(t, true)

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x")
	out := stdout + stderr
	if err != nil {
		t.Fatalf("a surviving launch must succeed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "tmux attach -t review-AI-001-x") {
		t.Errorf("a surviving launch must still tell the caller how to watch it:\n%s", out)
	}
}

// The stderr file is a sibling of the transcript, never merged into it: the
// transcript is jsonl an auditor parses, and interleaved diagnostics break that.
func TestSpecReviewRedirectsStderrBesideTheTranscript(t *testing.T) {
	root := makeRepo(t)
	seedPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")

	prev := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/tmux", nil }
	t.Cleanup(func() { lookPath = prev })

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err != nil {
		t.Fatalf("spec review --dry-run: %v", err)
	}
	out := stdout + stderr
	if !strings.Contains(out, "review-transcript.jsonl.stderr") {
		t.Errorf("stderr must be captured to its own file:\n%s", out)
	}
	if strings.Contains(out, "2>&1") {
		t.Errorf("stderr must not be folded into the transcript pipe:\n%s", out)
	}
}
