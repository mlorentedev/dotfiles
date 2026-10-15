package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeHerdrRunner replaces the herdr runner for one test, answering from a
// table keyed on the first two args and recording each call.
func fakeHerdrRunner(t *testing.T, answers map[string]string, fail map[string]bool) *[]string {
	t.Helper()
	var calls []string
	prev := herdrRunner
	herdrRunner = func(name string, args ...string) ([]byte, error) {
		key := strings.Join(args[:min(2, len(args))], " ")
		calls = append(calls, strings.Join(args, " "))
		if fail[key] {
			return []byte(answers[key]), os.ErrInvalid
		}
		return []byte(answers[key]), nil
	}
	t.Cleanup(func() { herdrRunner = prev })
	return &calls
}

func herdrAnswers(running bool) map[string]string {
	state := `{"running":false}`
	if running {
		state = `{"running":true}`
	}
	return map[string]string{
		"status server": state,
		"agent get":     `{"error":{"code":"agent_not_found","message":"not found"}}`,
		"worktree open": `{"result":{"root_pane":{"pane_id":"w3:p1"}}}`,
		"agent start":   `{"result":{}}`,
		"agent prompt":  `{"result":{}}`,
	}
}

func TestWorktreeAddStartsAnAgentInHerdr(t *testing.T) {
	repo := worktreeTestRepo(t)
	prompt := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(prompt, []byte("Implement #1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	calls := fakeHerdrRunner(t, herdrAnswers(true), map[string]bool{"agent get": true})
	out, err := executeWorktree(t, "add", "demo", "--repo", repo, "--agent", "claude", "--prompt-file", prompt)
	if err != nil {
		t.Fatalf("add --agent: %v\n%s", err, out)
	}
	wt := rootWorktree("demo")
	want := []string{
		"status server --json",
		"agent get feat-demo",
		"worktree open --cwd " + repo + " --path " + wt + " --no-focus",
		"agent start feat-demo --kind claude --pane w3:p1",
		"agent prompt feat-demo Implement #1",
	}
	if strings.Join(*calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("herdr calls =\n%s\nwant\n%s", strings.Join(*calls, "\n"), strings.Join(want, "\n"))
	}
	if !strings.Contains(out, "feat-demo") {
		t.Errorf("add does not name the agent it started:\n%s", out)
	}
}

func TestWorktreeAddRefusesBeforeCreatingWhenHerdrIsDown(t *testing.T) {
	repo := worktreeTestRepo(t)
	calls := fakeHerdrRunner(t, herdrAnswers(false), nil)
	_, err := executeWorktree(t, "add", "demo", "--repo", repo, "--agent", "claude")
	if err == nil || !strings.Contains(err.Error(), "no herdr server is running") {
		t.Fatalf("want a server-down refusal, got %v", err)
	}
	if _, err := os.Stat(rootWorktree("demo")); !os.IsNotExist(err) {
		t.Errorf("the worktree was created before the refusal (stat err %v)", err)
	}
	if len(*calls) != 1 {
		t.Errorf("herdr calls after the refusal: %q", *calls)
	}
}

func TestWorktreeAddRefusesAPromptWithoutAnAgent(t *testing.T) {
	repo := worktreeTestRepo(t)
	if _, err := executeWorktree(t, "add", "demo", "--repo", repo, "--prompt-file", "x"); err == nil || !strings.Contains(err.Error(), "--agent") {
		t.Fatalf("want a refusal naming --agent, got %v", err)
	}
}
