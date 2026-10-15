package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func runLane(t *testing.T, args ...string) error {
	t.Helper()
	root := New("test", "test")
	root.SetArgs(append([]string{"lane"}, args...))
	root.SetOut(new(strings.Builder))
	root.SetErr(new(strings.Builder))
	return root.Execute()
}

// exec reads the prompt from the file and hands the launch command to the exec
// seam; the prompt's own newlines and quotes survive as one argument.
func TestLaneExec_HandsTheAgentTheFilePromptAsOneArgument(t *testing.T) {
	var got []string
	prev := laneExec
	laneExec = func(argv []string) error { got = argv; return nil }
	t.Cleanup(func() { laneExec = prev })
	f := filepath.Join(t.TempDir(), "p.md")
	if err := os.WriteFile(f, []byte("Bug \"2501\"\nsecond line $HOME\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runLane(t, "exec", "--agent", "agy", "--prompt-file", f); err != nil {
		t.Fatal(err)
	}
	if want := []string{"agy", "-i", "Bug \"2501\"\nsecond line $HOME"}; !reflect.DeepEqual(got, want) {
		t.Errorf("exec argv = %q, want %q", got, want)
	}
	if err := runLane(t, "exec", "--agent", "codex", "--prompt-file", f); err == nil || !strings.Contains(err.Error(), "agy, claude, pi") {
		t.Errorf("an unknown agent is refused naming the known ones, got %v", err)
	}
	if err := runLane(t, "exec", "--agent", "claude", "--prompt-file", filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("a missing prompt file is an error")
	}
}
