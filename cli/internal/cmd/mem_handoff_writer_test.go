package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AC2 of MEMORY-009 (#1690): a fork is the one outcome where the handoff does
// not land where the writer asked, so the command says so, naming the key and
// both agents. The kept block is the other agent's, untouched.
func TestMemHandoffWriteAnnouncesAFork(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	pi := "### thread: master@msi (writer: pi)\n\n**Next action:** pi's step.\n"
	if err := os.WriteFile(memory, []byte("# M\n\n## Session Handoff\n\n"+pi), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	cmd := newMemCmd()
	cmd.SetArgs([]string{"handoff-write", "--memory", memory, "--thread", "master@msi", "--agent", "claude"})
	cmd.SetIn(bytes.NewBufferString("**Next action:** claude's step.\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("handoff-write --agent: %v", err)
	}
	for _, want := range []string{`"master@msi"`, "pi", "claude", `"master@msi+claude"`} {
		if !strings.Contains(errOut.String(), want) {
			t.Errorf("stderr does not name %s:\n%s", want, errOut.String())
		}
	}
	if !strings.Contains(out.String(), `"master@msi+claude"`) {
		t.Errorf("stdout does not report the thread written:\n%s", out.String())
	}
	after, err := os.ReadFile(memory)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), pi) || !strings.Contains(string(after), "### thread: master@msi+claude (writer: claude)") {
		t.Errorf("the file does not hold both blocks:\n%s", after)
	}
}

// Without --agent the command's words are today's: no fork notice, the key asked
// for, the same stdout line.
func TestMemHandoffWriteWithoutAgentSaysWhatItAlwaysSaid(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	if err := os.WriteFile(memory, []byte("# M\n\n## Session Handoff\n\n### thread: master@msi (writer: pi)\n\nold\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	cmd := newMemCmd()
	cmd.SetArgs([]string{"handoff-write", "--memory", memory, "--thread", "master@msi"})
	cmd.SetIn(bytes.NewBufferString("new\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr is not empty:\n%s", errOut.String())
	}
	if want := "wrote      thread \"master@msi\" in " + memory + "\n"; out.String() != want {
		t.Errorf("stdout = %q, want %q", out.String(), want)
	}
}
