package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

// HARNESS-201. Lists union on merge, so narrowing a deny glob in the template
// does not remove the old, broader glob from a settings.json already deployed:
// the old `git push *--force*` would keep denying the lease form on every
// existing box. What removes it is the deploy record, and only where one exists.
// This drives the real template through the real merge, the way the owner's box
// receives it, rather than reading the template's text.

const oldPushDeny = `{"permissions":{"deny":["Bash(rm -rf /:*)","Bash(git push *--force*)","Bash(git push -f*)","Bash(git push * -f*)"]}}`

func claudeSettingsConfig() Config {
	return Config{Name: "claude-settings", Src: "ai/claude/settings.json", Dst: "{HOME}/.claude/settings.json", Mode: "0644", Strategy: StrategyMerge}
}

func repoWithClaudeTemplate(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, "ai", "claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func denyList(t *testing.T, dst string) []string {
	t.Helper()
	var out []string
	perms, _ := readObject(t, dst)["permissions"].(map[string]any)
	list, _ := perms["deny"].([]any)
	for _, r := range list {
		s, _ := r.(string)
		out = append(out, s)
	}
	return out
}

func has(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func realClaudeTemplate(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "ai", "claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestClaudeTemplate_RetiresTheBroadForceGlobOnAnExistingBox(t *testing.T) {
	const oldBroad = "Bash(git push *--force*)"
	root := repoWithClaudeTemplate(t, oldPushDeny)
	home, records := t.TempDir(), t.TempDir()
	c := claudeSettingsConfig()
	dst := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(`{"permissions":{"deny":["Bash(box-only:*)"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	mergeStep(t, c, root, home, records, oldPushDeny)
	if !has(denyList(t, dst), oldBroad) {
		t.Fatalf("setup: the old template did not deploy %q", oldBroad)
	}

	mergeStep(t, c, root, home, records, realClaudeTemplate(t))
	got := denyList(t, dst)
	if has(got, oldBroad) {
		t.Errorf("the old broad glob %q survived the new template: the lease form stays denied on this box\n%q", oldBroad, got)
	}
	for _, want := range []string{"Bash(git push *--force *)", "Bash(npm run land:push*)", "Bash(box-only:*)", "Bash(rm -rf /:*)"} {
		if !has(got, want) {
			t.Errorf("%q missing after the deploy: %q", want, got)
		}
	}
}

// Without a record the old glob stays: a deploy cannot tell what the repo wrote
// from what the owner added. The ADR names this; the test keeps it true to life.
func TestClaudeTemplate_WithoutARecordTheBroadGlobStays(t *testing.T) {
	root := repoWithClaudeTemplate(t, oldPushDeny)
	home := t.TempDir()
	c := claudeSettingsConfig()
	dst := filepath.Join(home, ".claude", "settings.json")

	mergeStep(t, c, root, home, "", oldPushDeny)
	mergeStep(t, c, root, home, "", realClaudeTemplate(t))
	if !has(denyList(t, dst), "Bash(git push *--force*)") {
		t.Error("a deploy with no record now retires the old glob: update the ADR rollout note, then change this test to say so")
	}
}
