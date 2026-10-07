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

// --agent names the record (MEMORY-015, #1929), and a refused one is reported
// on stderr while the hook still exits 0: a misconfigured hook must say so, not
// leave every session of that harness without a record in silence.
func TestMemSessionEnd_AgentFlag(t *testing.T) {
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
	for _, tc := range []struct{ agent, glob, stderr string }{
		{"opencode", "*-proj-opencode.md", ""},
		{"Not One Word", "*-proj-*.md", "one lower-case word"},
	} {
		var stderr bytes.Buffer
		cmd := newMemCmd()
		cmd.SetArgs([]string{"session-end", "--agent", tc.agent})
		cmd.SetIn(bytes.NewBufferString(`{"cwd":"/x/proj","session_id":"s1"}`))
		cmd.SetOut(io.Discard)
		cmd.SetErr(&stderr)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%s: session-end must exit 0, got %v", tc.agent, err)
		}
		matches, _ := filepath.Glob(filepath.Join(vault, "10_projects", "proj", "sessions", tc.glob))
		if tc.stderr == "" && len(matches) != 1 {
			t.Errorf("%s: want one record matching %s, got %v", tc.agent, tc.glob, matches)
		}
		// A refused agent writes nothing: the opencode record from the case
		// before is the only one there.
		if all, _ := filepath.Glob(filepath.Join(vault, "10_projects", "proj", "sessions", "*.md")); tc.stderr != "" && len(all) != 1 {
			t.Errorf("%s: a refused agent must write no record, got %v", tc.agent, all)
		}
		if !strings.Contains(stderr.String(), tc.stderr) {
			t.Errorf("%s: stderr %q lacks %q", tc.agent, stderr.String(), tc.stderr)
		}
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

// #1606 end to end: run from the vault checkout with a project's MEMORY.md and no
// --thread, handoff-write must refuse and leave every thread byte-identical,
// including the one whose key the vault's branch would have produced.
func TestMemHandoffWriteFromAnotherRepositoryTouchesNoThread(t *testing.T) {
	vault := filepath.Join(t.TempDir(), "knowledge")
	if err := os.MkdirAll(filepath.Join(vault, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vault, ".git", "HEAD"), []byte("ref: refs/heads/master\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	memory := filepath.Join(vault, "10_projects", "dotfiles", "memory", "MEMORY.md")
	if err := os.MkdirAll(filepath.Dir(memory), 0o755); err != nil {
		t.Fatal(err)
	}
	host, _ := os.Hostname()
	host, _, _ = strings.Cut(strings.ToLower(host), ".")
	before := "# M\n\n## Session Handoff\n\n### thread: master@" + host + "\n\nanother session's block\n"
	if err := os.WriteFile(memory, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(vault)

	cmd := newMemCmd()
	cmd.SetArgs([]string{"handoff-write", "--memory", memory})
	cmd.SetIn(bytes.NewBufferString("**Next action:** this session's handoff\n"))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err == nil {
		t.Error("handoff-write keyed a dotfiles thread from the vault's branch; want a refusal naming --thread")
	}
	after, err := os.ReadFile(memory)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != before {
		t.Errorf("MEMORY.md changed:\n--- got ---\n%s\n--- want ---\n%s", after, before)
	}
}
