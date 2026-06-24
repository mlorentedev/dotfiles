package mem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)

// writeMemory drops a MEMORY.md for project "proj" under vault root and returns
// the project cwd the hook payload would carry.
func writeMemory(t *testing.T, vault, content string) {
	t.Helper()
	dir := filepath.Join(vault, "10_projects", "proj", "memory")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSessionEnd_NoOps covers the resilience contract: every trivial / missing /
// malformed input is a clean no-op (no file written, no error surfaced).
func TestSessionEnd_NoOps(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		memory  string // MEMORY.md content; "" means do not create the file
	}{
		{"empty stdin", "", ""},
		{"malformed json", "{not json", ""},
		{"missing cwd", `{"session_id":"abc"}`, ""},
		{"absent memory file", `{"cwd":"/x/proj","session_id":"abc"}`, ""},
		{"no handoff heading", `{"cwd":"/x/proj","session_id":"abc"}`, "# Project\n\nsome notes\n"},
		{"whitespace-only block", `{"cwd":"/x/proj","session_id":"abc"}`, "# P\n\n## Session Handoff\n\n   \n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vault := t.TempDir()
			if tc.memory != "" {
				writeMemory(t, vault, tc.memory)
			}
			written, err := SessionEnd([]byte(tc.payload), vault, fixedNow)
			if err != nil {
				t.Fatalf("expected no error on no-op, got %v", err)
			}
			if written != "" {
				t.Fatalf("expected no file written, got %q", written)
			}
		})
	}
}

// TestSessionEnd_HappyPath archives the handoff block into a durable record.
func TestSessionEnd_HappyPath(t *testing.T) {
	vault := t.TempDir()
	writeMemory(t, vault, "# Project\n\n## Index\n\n- foo\n\n## Session Handoff\n\n**Last task:** shipped PR1.\n**Next:** PR2.\n")

	written, err := SessionEnd([]byte(`{"cwd":"/home/me/proj","session_id":"sid-123"}`), vault, fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(vault, "10_projects", "proj", "sessions", "2026-06-23-proj-claude.md")
	if written != want {
		t.Fatalf("written path = %q, want %q", written, want)
	}
	b, err := os.ReadFile(written)
	if err != nil {
		t.Fatalf("reading record: %v", err)
	}
	got := string(b)
	for _, frag := range []string{
		`id: "session-2026-06-23-proj-claude"`,
		"type: session",
		"session_id: sid-123",
		"agent: claude",
		"project: proj",
		`date: "2026-06-23"`,
		"tags: [session, handoff, proj]",
		"# Session 2026-06-23 — proj (claude)",
		"**Last task:** shipped PR1.",
		"**Next:** PR2.",
	} {
		if !strings.Contains(got, frag) {
			t.Errorf("record missing %q\n--- got ---\n%s", frag, got)
		}
	}
	// The block stops at the next "## " heading — the index must not leak in.
	if strings.Contains(got, "## Index") || strings.Contains(got, "- foo") {
		t.Errorf("record leaked content past the handoff block:\n%s", got)
	}
}

// TestSessionEnd_DefaultsSessionID falls back to "unknown" when the payload omits it.
func TestSessionEnd_DefaultsSessionID(t *testing.T) {
	vault := t.TempDir()
	writeMemory(t, vault, "## Session Handoff\n\nwork happened\n")
	written, err := SessionEnd([]byte(`{"cwd":"/x/proj"}`), vault, fixedNow)
	if err != nil || written == "" {
		t.Fatalf("expected a record, got written=%q err=%v", written, err)
	}
	b, _ := os.ReadFile(written)
	if !strings.Contains(string(b), "session_id: unknown") {
		t.Errorf("expected session_id: unknown, got:\n%s", b)
	}
}

// TestSessionEnd_EmptyVaultIsNoOp: an unresolved vault ("") never writes.
func TestSessionEnd_EmptyVaultIsNoOp(t *testing.T) {
	written, err := SessionEnd([]byte(`{"cwd":"/x/proj","session_id":"abc"}`), "", fixedNow)
	if err != nil || written != "" {
		t.Fatalf("empty vault must be a no-op, got written=%q err=%v", written, err)
	}
}
