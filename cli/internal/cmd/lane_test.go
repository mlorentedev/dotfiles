package cmd

import (
	"errors"
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

// next wires git, gh and herdr to the hand-off: from a repo with a lane file it
// names the first free issue and, under --dry-run, starts nothing.
func TestLaneNext_DryRunNamesTheNextIssueAndChangesNothing(t *testing.T) {
	top := t.TempDir()
	if err := os.MkdirAll(filepath.Join(top, ".dotf"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		".dotf/lane.json":      `{"queue":[5,6],"template":".dotf/lane-prompt.md"}`,
		".dotf/lane-prompt.md": "Do #{{.Number}}",
	} {
		if err := os.WriteFile(filepath.Join(top, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var asked []string
	prev := laneRun
	laneRun = func(name string, args ...string) (string, error) {
		line := name + " " + strings.Join(args, " ")
		asked = append(asked, line)
		switch {
		case strings.HasPrefix(line, "git rev-parse --show-toplevel"):
			return top + "\n", nil
		case strings.HasPrefix(line, "git rev-parse --path-format=absolute --git-common-dir"):
			return filepath.Join(top, ".git") + "\n", nil
		case strings.HasPrefix(line, "gh issue view 5"):
			return `{"number":5,"state":"OPEN","title":"BUG-5: the first","assignees":[]}`, nil
		case strings.HasPrefix(line, "herdr pane current"):
			return `{"result":{"pane":{"agent":"claude"}}}`, nil
		case strings.HasPrefix(line, "herdr worktree list"):
			return `{"result":{"worktrees":[]}}`, nil
		}
		return "", errors.New("unexpected: " + line)
	}
	t.Cleanup(func() { laneRun = prev })
	root := New("test", "test")
	var out strings.Builder
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"lane", "next", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "next: #5 BUG-5: the first -> fix/the-first (claude)") {
		t.Errorf("output:\n%s", out.String())
	}
	for _, a := range asked {
		if strings.Contains(a, "create") || strings.Contains(a, "pane run") || strings.Contains(a, "fetch") {
			t.Errorf("a dry run asked %q", a)
		}
	}
}
