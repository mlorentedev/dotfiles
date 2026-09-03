package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// executeStdin runs the CLI with a stdin payload. The from-hook mode reads its
// payload from stdin by contract (AC5), so it cannot be exercised by execute().
func executeStdin(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := New("dev", "")
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errBuf.String(), err
}

// repoRootForTest is shared with harness_resolve_tier_test.go — it walks up to
// the checkout root rather than assuming a fixed depth.

func TestSuggestFromHookReadsStdin(t *testing.T) {
	root := repoRootForTest(t)
	payload := `{"hook_event_name":"UserPromptSubmit","prompt":"add tests for this and use TDD"}`

	stdout, stderr, err := executeStdin(t, payload, "harness", "suggest", "--from-hook", "--repo-root", root)
	if err != nil {
		t.Fatalf("from-hook must never error: %v (stderr %s)", err, stderr)
	}
	if !strings.Contains(stdout, "persona") {
		t.Errorf("expected a persona suggestion, got:\n%s", stdout)
	}

	// AC5: the prompt arrives on stdin and nowhere else. A --prompt flag would be
	// a shell-quoted user prompt on a command line, which is an injection
	// surface, so the from-hook path must not consult one.
	stdout2, _, err := executeStdin(t, payload, "harness", "suggest", "--from-hook",
		"--repo-root", root, "--prompt", "completely unrelated kubernetes helm chart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout2 != stdout {
		t.Errorf("--prompt changed from-hook output; stdin must be the only source\ngot:\n%s\nwant:\n%s", stdout2, stdout)
	}
}

func TestSuggestFromHookRecordsPromptField(t *testing.T) {
	root := repoRootForTest(t)

	// AC6 at the command layer: the field the prompt arrived under is reported,
	// so a guess about an undocumented payload becomes a measurement the next
	// session can read.
	_, stderr, err := executeStdin(t, `{"user_prompt":"add tests for this and use TDD"}`,
		"harness", "suggest", "--from-hook", "--repo-root", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "user_prompt") {
		t.Errorf("the arriving field name must be recorded, stderr was:\n%s", stderr)
	}

	// An unrecognised payload records THAT, rather than passing silently.
	_, stderr2, err := executeStdin(t, `{"session_id":"abc"}`,
		"harness", "suggest", "--from-hook", "--repo-root", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(stderr2), "unrecognised") {
		t.Errorf("an unrecognised payload must say so, stderr was:\n%s", stderr2)
	}
}

// TestSuggestFromHookNeverExitsNonZero is AC7, and it is the data-loss guard.
//
// On UserPromptSubmit, exit code 2 is documented verbatim as "Blocks prompt
// processing and erases the prompt". That is strictly worse than the gate's
// worst case, which is a refused tool call. Every branch that can return an
// error is a row here, asserted rather than inspected.
func TestSuggestFromHookNeverExitsNonZero(t *testing.T) {
	root := repoRootForTest(t)

	empty := t.TempDir() // no harness/ at all: the non-dotfiles-repo case
	brokenRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(brokenRoot, "harness", "agents", "builder"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brokenRoot, "harness", "triggers.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brokenRoot, "harness", "agents", "builder", "AGENT.md"),
		[]byte("---\nname: builder\nskills: [\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		stdin   string
		rootDir string
	}{
		{"malformed json payload", `{"prompt":`, root},
		{"payload is an array", `["prompt"]`, root},
		{"empty payload", ``, root},
		{"no prompt field", `{"session_id":"abc"}`, root},
		{"blank prompt", `{"prompt":"   "}`, root},
		{"prompt matching no rule", `{"prompt":"zzzz nothing matches this at all"}`, root},
		{"harness root missing entirely", `{"prompt":"add tests"}`, empty},
		{"triggers.json unparseable", `{"prompt":"add tests"}`, brokenRoot},
		{"persona record unreadable", `{"prompt":"add tests"}`, brokenRoot},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, err := executeStdin(t, tc.stdin, "harness", "suggest", "--from-hook", "--repo-root", tc.rootDir)
			if err != nil {
				t.Errorf("exit must be 0 — a non-zero exit here ERASES THE USER'S PROMPT. got %v (stderr %s)", err, stderr)
			}
		})
	}
}
