package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MEMORY-008 slice 1: a body without a Next action, or with a label outside the
// canonical set, is named on stderr and still written. Refusing it would lose a
// handoff over its shape, which is the loss this command exists to prevent.
func TestMemHandoffWriteNamesWhatTheBodyLacksAndStillWrites(t *testing.T) {
	memory := filepath.Join(t.TempDir(), "MEMORY.md")
	if err := os.WriteFile(memory, []byte("# M\n\n## Session Handoff\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	cmd := newMemCmd()
	cmd.SetArgs([]string{"handoff-write", "--memory", memory, "--thread", "feat-x"})
	cmd.SetIn(bytes.NewBufferString("**Last task:** x.\n**Trap:** the vault auto-commits.\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("handoff-write refused a body over its shape: %v", err)
	}
	for _, want := range []string{"Next action", `"Trap"`} {
		if !strings.Contains(errOut.String(), want) {
			t.Errorf("stderr does not name %s:\n%s", want, errOut.String())
		}
	}
	if after, _ := os.ReadFile(memory); !strings.Contains(string(after), "**Trap:** the vault auto-commits.") {
		t.Errorf("the body was not written:\n%s", after)
	}
}
