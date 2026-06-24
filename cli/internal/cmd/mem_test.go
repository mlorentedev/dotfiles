package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
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
