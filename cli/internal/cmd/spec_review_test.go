package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
