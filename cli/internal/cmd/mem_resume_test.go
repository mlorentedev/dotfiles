package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runMemResumeT(t *testing.T, args ...string) string {
	t.Helper()
	cmd := newMemCmd()
	cmd.SetArgs(append([]string{"resume"}, args...))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// handoff-write stores a thread and resume renders it back from the file.
func TestMemResumeRendersWhatHandoffWriteStored(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	if err := os.WriteFile(memory, []byte("# M\n\n## Session Handoff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runHandoffWrite(t, memory, "**Decisions:** only in the file.\n**Next action:** next.\n"); err != nil {
		t.Fatal(err)
	}
	out := runMemResumeT(t, "--memory", memory, "--thread", "feat-x", "--format", "context")
	if !strings.Contains(out, "only in the file") || !strings.Contains(out, filepath.Join("threads", "feat-x.md")) {
		t.Errorf("resume did not render the stored thread:\n%s", out)
	}
}

func TestMemResumeListsCandidates(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	doc := "# M\n\n## Session Handoff\n\n### thread: feat-a (writer: pi)\n\n**Next action:** a.\n"
	if err := os.WriteFile(memory, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runMemResumeT(t, "--memory", memory, "--thread", "feat-a", "--agent", "claude")
	if !strings.Contains(out, "No single thread") || !strings.Contains(out, "dotf mem resume --thread feat-a --memory '"+memory+"' --agent claude\n") {
		t.Errorf("want the candidates listed:\n%s", out)
	}
}
