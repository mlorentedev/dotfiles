package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMemSessionEnd_WritesRecordAndExitsZero exercises the full stdin -> resolve
// -> write wiring through the cobra command (VAULT_PATH short-circuits the
// resolver cascade).
func TestMemSessionEnd_WritesRecordAndExitsZero(t *testing.T) {
	vault := t.TempDir()
	t.Setenv("VAULT_PATH", vault)
	memDir := filepath.Join(vault, "10_projects", "proj", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"),
		[]byte("## Session Handoff\n\nshipped it\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newMemCmd()
	cmd.SetArgs([]string{"session-end"})
	cmd.SetIn(bytes.NewBufferString(`{"cwd":"/x/proj","session_id":"s1"}`))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("session-end must exit 0, got %v", err)
	}

	matches, _ := filepath.Glob(filepath.Join(vault, "10_projects", "proj", "sessions", "*-proj-claude.md"))
	if len(matches) != 1 {
		t.Fatalf("expected exactly 1 session record, got %v", matches)
	}
}

// TestMemSessionEnd_MalformedInputExitsZero pins the resilience contract: even
// garbage on stdin must never crash the session (exit 0, no file).
func TestMemSessionEnd_MalformedInputExitsZero(t *testing.T) {
	vault := t.TempDir()
	t.Setenv("VAULT_PATH", vault)

	cmd := newMemCmd()
	cmd.SetArgs([]string{"session-end"})
	cmd.SetIn(bytes.NewBufferString("garbage not json"))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("malformed input must still exit 0, got %v", err)
	}
}

// TestMemProjectKey exercises `dotf mem project-key <path>`: the single source of
// the Claude auto-memory encoding that the PowerShell twins call instead of
// re-implementing it (BUG-031/#689). Output must equal memlink.ClaudeProjectKey.
func TestMemProjectKey(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"windows path", `C:\Users\me\p`, "C--Users-me-p"},
		{"posix path", "/home/me/p", "-home-me-p"},
		{"dotted repo name", "/home/me/svqtriana.github.io", "-home-me-svqtriana-github-io"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newMemCmd()
			var out bytes.Buffer
			cmd.SetArgs([]string{"project-key", tc.in})
			cmd.SetOut(&out)
			cmd.SetErr(io.Discard)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("project-key %q: unexpected error %v", tc.in, err)
			}
			if got := strings.TrimSpace(out.String()); got != tc.want {
				t.Errorf("project-key %q = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// #1651: --dry-run must show the migration of the un-threaded block, and the
// command names it on stderr, because nothing else tells the operator that a
// block they did not write moved.
func TestMemHandoffWriteDryRunShowsTheLegacyMigration(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	before := "# M\n\n## Session Handoff\n> Updated: 2026-09-05  \n**Next action:** merge #412.\n\n### thread: master@msi\n\nlive\n"
	if err := os.WriteFile(memory, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	cmd := newMemCmd()
	cmd.SetArgs([]string{"handoff-write", "--memory", memory, "--thread", "feat-x", "--dry-run"})
	cmd.SetIn(bytes.NewBufferString("**Next action:** new work.\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("handoff-write --dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "### thread: legacy-2026-09-05") {
		t.Errorf("--dry-run does not show the migration:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "legacy-2026-09-05") {
		t.Errorf("stderr does not name the migrated thread:\n%s", errOut.String())
	}
	if after, _ := os.ReadFile(memory); string(after) != before {
		t.Error("--dry-run wrote the file")
	}
}
