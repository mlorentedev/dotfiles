package harness

import (
	"path/filepath"
	"strings"
	"testing"
)

// HARNESS-201. The guard rides the same PreToolUse event as the persona gate and
// as Orca's hook. isOurs claims any command containing "dotf harness gate" for
// the id `gate`, and an event carrying two of our hooks needs per-purpose
// identity, or the second silently evicts the first. This runs the shipped
// manifest through the real emission and merge over a settings document that
// already holds a foreign PreToolUse group.
func TestPushGuardAndGateCoexistOnPreToolUse(t *testing.T) {
	targets, err := LoadBindTargets(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var claude *BindTarget
	for i := range targets {
		if targets[i].Agent == "claude" {
			claude = &targets[i]
		}
	}
	if claude == nil {
		t.Fatal("no claude bind target in the manifest")
	}
	cmds, err := claude.HookCommands("/opt/bin/dotf")
	if err != nil {
		t.Fatal(err)
	}
	var pre []HookCommand
	for _, c := range cmds {
		if c.Event == "PreToolUse" {
			pre = append(pre, c)
		}
	}
	ids := map[string]string{}
	for _, c := range pre {
		ids[c.ID] = c.Command
	}
	if !strings.HasSuffix(ids["push-guard"], " harness push-guard") || !strings.Contains(ids["gate"], "harness gate") {
		t.Fatalf("PreToolUse hooks = %v, want both gate and push-guard", ids)
	}
	if strings.Contains(ids["push-guard"], "dotf harness gate") {
		t.Fatalf("push-guard's command %q matches the gate's signature and would be adopted as the gate", ids["push-guard"])
	}

	doc := decode(t, deployedClaudeSettings)
	merged, changed, err := MergeHooks(doc, cmds)
	if err != nil || !changed {
		t.Fatalf("MergeHooks: changed=%v err=%v", changed, err)
	}
	groups := merged["hooks"].(map[string]any)["PreToolUse"].([]any)
	var commands []string
	for _, g := range groups {
		for _, h := range g.(map[string]any)["hooks"].([]any) {
			commands = append(commands, h.(map[string]any)["command"].(string))
		}
	}
	joined := strings.Join(commands, "\n")
	for _, want := range []string{"claude-hook.sh", "harness gate", "harness push-guard"} {
		if !strings.Contains(joined, want) {
			t.Errorf("PreToolUse lost %q: %v", want, commands)
		}
	}
	again, changed, err := MergeHooks(merged, cmds)
	if err != nil || changed {
		t.Fatalf("second merge: changed=%v err=%v, want a no-op", changed, err)
	}
	if len(again["hooks"].(map[string]any)["PreToolUse"].([]any)) != len(groups) {
		t.Error("a second merge accumulated a group")
	}
}
