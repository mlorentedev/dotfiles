package converge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bindEnv is a checkout whose manifest binds two hooks into claude's settings
// file, declares a target that does not emit and one whose harness is not
// installed, and a HOME whose settings file already carries a foreign hook.
func bindEnv(t *testing.T) Env {
	t.Helper()
	repo := t.TempDir()
	writeFixture(t, repo, map[string]string{
		"harness/manifest.json": `{"agents": {"bind": [
		  {"agent": "claude", "file": ".claude/settings.json", "format": "command-hook", "matcher": true,
		   "emit_hooks": [
		     {"id": "mem-session-start", "event": "SessionStart", "command": "mem session-start", "timeout": 30},
		     {"id": "gate", "event": "PreToolUse", "command": "harness gate --harness claude", "timeout": 5}
		   ]},
		  {"agent": "pi", "file": ".pi/agent/extensions/dotfiles-gate.ts", "format": "ts-extension", "emit": false}
		], "bind_named": [
		  {"agent": "agy", "file": ".gemini/config/hooks.json", "format": "hooks-json", "requires_command": "agy",
		   "emit_hooks": [{"id": "gate", "event": "PreInvocation", "command": "harness gate --harness agy"}]}
		]}}`,
	})
	home := t.TempDir()
	writeFixture(t, home, map[string]string{
		".claude/settings.json": `{"model": "opus", "hooks": {"SessionStart": [
		  {"matcher": "", "hooks": [{"type": "command", "command": "/opt/orca/hook.sh"}]}
		]}}`,
	})
	return Env{RepoRoot: repo, Home: home, GOOS: "darwin"}
}

func newRecordsBind() recordsBind {
	return recordsBind{
		dotf: func(home string) string { return filepath.Join(home, ".local", "bin", "dotf") },
		has:  func(string) bool { return false },
	}
}

func TestRecordsBind_PlanWritesNothingApplyConvergesAndRerunIsANoOp(t *testing.T) {
	env := bindEnv(t)
	r := newRecordsBind()
	settings := filepath.Join(env.Home, ".claude", "settings.json")
	before, _ := os.ReadFile(settings)

	plan, err := r.Reconcile(env, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changes != 1 || !strings.Contains(plan.Detail, "to write: .claude/settings.json") {
		t.Errorf("plan: want 1 change naming the claude settings, got %d (%s)", plan.Changes, plan.Detail)
	}
	for _, want := range []string{"pi (declared emit:false (ts-extension))", "agy (agy is not installed)"} {
		if !strings.Contains(plan.Detail, want) {
			t.Errorf("plan detail %q does not name the skipped target %q", plan.Detail, want)
		}
	}
	if after, _ := os.ReadFile(settings); string(after) != string(before) {
		t.Fatal("the plan wrote the settings file")
	}
	if err := r.Probe(env); err == nil {
		t.Error("the probe passed before the hooks were bound")
	}

	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	if err := r.Probe(env); err != nil {
		t.Fatalf("probe after apply: %v", err)
	}
	again, err := r.Reconcile(env, true)
	if err != nil || again.Changes != 0 {
		t.Errorf("second plan: want 0 changes, got %d (%s, %v)", again.Changes, again.Detail, err)
	}

	doc := map[string]any{}
	raw, _ := os.ReadFile(settings)
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["model"] != "opus" || !strings.Contains(string(raw), "/opt/orca/hook.sh") {
		t.Errorf("the bind lost a key or a hook it does not own:\n%s", raw)
	}
}

// #2232: another writer of the settings file kept our hooks and dropped the
// markers. A converge must read that as converged, not rewrite the file on every
// run for the next writer to undo.
func TestRecordsBind_HooksWithStrippedMarkersAreConverged(t *testing.T) {
	env := bindEnv(t)
	r := newRecordsBind()
	if _, err := r.Reconcile(env, false); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(env.Home, ".claude", "settings.json")
	raw, _ := os.ReadFile(settings)
	stripped := strings.ReplaceAll(string(raw), `"_managed"`, `"_dropped"`)
	stripped = strings.NewReplacer(`"_dropped": "dotfiles-harness:mem-session-start",`, "",
		`"_dropped": "dotfiles-harness:gate",`, "").Replace(stripped)
	if strings.Contains(stripped, "dotfiles-harness") {
		t.Fatalf("fixture did not strip the markers:\n%s", stripped)
	}
	if err := os.WriteFile(settings, []byte(stripped), 0o600); err != nil {
		t.Fatal(err)
	}

	plan, err := r.Reconcile(env, true)
	if err != nil || plan.Changes != 0 {
		t.Errorf("hooks without markers: want 0 changes, got %d (%s, %v)", plan.Changes, plan.Detail, err)
	}
}

// Unwired, the step is reported skipped with the reason, never as converged,
// and never guesses which binary the hooks should name.
func TestRecordsBind_NoResolverWiredIsSkippedNotPassed(t *testing.T) {
	env := bindEnv(t)
	res, err := (recordsBind{}).Reconcile(env, false)
	if err != nil || res.Skip != noResolver {
		t.Fatalf("want a skip naming the missing resolver, got %+v, %v", res, err)
	}
	if raw, _ := os.ReadFile(filepath.Join(env.Home, ".claude", "settings.json")); strings.Contains(string(raw), "dotf") {
		t.Errorf("an unwired step wrote hooks:\n%s", raw)
	}
}
