package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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
	laneNextDryRun(t, t.TempDir())
}

// A checkout under a path with a space (C:\Users\Jane Doe\...) must hand the
// lane over like any other (#2537): the command typed into the pane carries no
// path of the repository's.
func TestLaneNext_ARepositoryUnderAPathWithASpaceHandsTheLaneOver(t *testing.T) {
	laneNextDryRun(t, filepath.Join(t.TempDir(), "Jane Doe", "my repo"))
}

func laneNextDryRun(t *testing.T, top string) {
	t.Helper()
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
		case strings.Contains(line, " ls-remote origin refs/dotf/claims/5"):
			return "", nil // nobody holds #5
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
	// The claim store is wired into `lane next`: a dry run asks the remote
	// whether #5 is held. Dropping the wiring leaves every other assertion here
	// green, so this one is the guard (lesson 457).
	if !slices.ContainsFunc(asked, func(a string) bool { return strings.Contains(a, " ls-remote origin refs/dotf/claims/5") }) {
		t.Errorf("lane next never asked the remote about the claim on #5; asked:\n%s", strings.Join(asked, "\n"))
	}
	for _, a := range asked {
		if strings.Contains(a, "create") || strings.Contains(a, "pane run") || strings.Contains(a, "fetch") || strings.Contains(a, " push ") {
			t.Errorf("a dry run asked %q", a)
		}
	}
}

// A refusal reaches the terminal: with the error silenced, a successor's pane
// died at a blank prompt (exit 1, nothing printed). The text a person needs is
// the one the error carries, so assert it on stderr and not only in the return.
func TestLane_RefusalsAreShownOnStderr(t *testing.T) {
	f := filepath.Join(t.TempDir(), "p.md")
	if err := os.WriteFile(f, []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"unknown agent":       {[]string{"exec", "--agent", "copilot", "--prompt-file", f}, `no launch command for agent "copilot"`},
		"known agents listed": {[]string{"exec", "--agent", "copilot", "--prompt-file", f}, "agy, claude, pi"},
		"missing prompt":      {[]string{"exec", "--agent", "claude", "--prompt-file", f + ".missing"}, "lane exec:"},
	} {
		root := New("test", "test")
		var stderr strings.Builder
		root.SetOut(new(strings.Builder))
		root.SetErr(&stderr)
		root.SetArgs(append([]string{"lane"}, tc.args...))
		if err := root.Execute(); err == nil {
			t.Fatalf("%s: want an error", name)
		}
		if !strings.Contains(stderr.String(), tc.want) {
			t.Errorf("%s: stderr = %q, want it to contain %q", name, stderr.String(), tc.want)
		}
	}
}

// The line `lane next` types names the prompt relative to the pane's directory.
// A pane standing in a subdirectory of the worktree does not hold the file, so
// readPrompt looks at the worktree root; a file that is nowhere stays an error.
func TestReadPrompt_FallsBackToTheWorktreeRoot(t *testing.T) {
	top, sub := t.TempDir(), filepath.Join(t.TempDir(), "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(top, "p.md"), []byte("from the root"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	prev := laneRun
	laneRun = func(name string, args ...string) (string, error) {
		if name == "git" && strings.Join(args, " ") == "rev-parse --show-toplevel" {
			return top + "\n", nil
		}
		return "", errors.New("unexpected: " + name)
	}
	t.Cleanup(func() { laneRun = prev })
	got, err := readPrompt("p.md")
	if err != nil || string(got) != "from the root" {
		t.Errorf("readPrompt(p.md) = %q, %v; want the root's file", got, err)
	}
	if _, err := readPrompt("missing.md"); err == nil {
		t.Error("a prompt that is nowhere is an error")
	}
	if err := os.WriteFile(filepath.Join(sub, "p.md"), []byte("from the cwd"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := readPrompt("p.md"); err != nil || string(got) != "from the cwd" {
		t.Errorf("a file in the cwd wins: %q, %v", got, err)
	}
}
