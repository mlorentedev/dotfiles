package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The judging cases run in tests/check-knowledge-gate.bats against the shell
// gate. These cover the command's own contract: exit codes for its inputs and
// the job summary.

func runKnowledgeGate(t *testing.T, args ...string) (string, int) {
	t.Helper()
	root := New("test", "")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"gate", "knowledge"}, args...))
	err := root.Execute()
	return out.String(), ExitCode(err)
}

func TestGateKnowledgeUsage(t *testing.T) {
	if _, code := runKnowledgeGate(t, "--event", "e.json"); code != 2 {
		t.Errorf("missing --base/--head: exit %d, want 2", code)
	}
}

func TestGateKnowledgeNoPRIsAWiringError(t *testing.T) {
	ev := filepath.Join(t.TempDir(), "e.json")
	if err := os.WriteFile(ev, []byte(`{"ref":"refs/heads/main"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out, code := runKnowledgeGate(t, "--event", ev, "--base", "main", "--head", "HEAD")
	if code != 2 || !strings.Contains(out, "no PR context") {
		t.Errorf("exit %d, output %q: want 2 and \"no PR context\"", code, out)
	}
}

// A body without the section fails before any diff, so this needs no repository.
func TestGateKnowledgeSummary(t *testing.T) {
	dir := t.TempDir()
	ev := filepath.Join(dir, "e.json")
	if err := os.WriteFile(ev, []byte(`{"pull_request":{"number":1,"body":"no section","user":{"login":"u"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := filepath.Join(dir, "summary.md")
	if err := os.WriteFile(sum, []byte("earlier\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, code := runKnowledgeGate(t, "--event", ev, "--base", "main", "--head", "HEAD", "--summary", sum); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	got, _ := os.ReadFile(sum)
	want := "earlier\n### knowledge-gate: failed\n\n- the PR body has no \"## Knowledge\" section\n\n"
	if string(got) != want {
		t.Errorf("summary = %q, want %q (appended, not replaced)", got, want)
	}
}

func runSpecGate(t *testing.T, args ...string) (string, int) {
	t.Helper()
	root := New("test", "")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"gate", "spec"}, args...))
	code := ExitCode(root.Execute())
	return out.String(), code
}

// The verdicts run in tests/check-spec-gate.bats and spec-gate-archive.bats
// against the script; these cover the command's own inputs.
func TestGateSpecWiringErrors(t *testing.T) {
	if _, code := runSpecGate(t, "--event", "e.json"); code != 2 {
		t.Errorf("missing --base/--head: exit %d, want 2", code)
	}
	ev := filepath.Join(t.TempDir(), "e.json")
	if err := os.WriteFile(ev, []byte(`{"ref":"refs/heads/main"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, code := runSpecGate(t, "--event", ev, "--base", "main", "--head", "HEAD"); code != 2 || !strings.Contains(out, "no PR context") {
		t.Errorf("exit %d, output %q: want 2 and \"no PR context\"", code, out)
	}
}
