package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// liveShapedSettings mirrors the DEPLOYED ~/.claude/settings.json measured on
// 2026-08-27, not a fixture built to pass.
//
// Two properties of the real file drive every assertion below and neither is
// obvious: our SessionStart hook carries NO marker (it was written by the
// positional jq writer, before markers existed), and SessionStart holds a SECOND
// group belonging to Orca. A fixture with one group, or with our entry already
// marked, would exercise neither the adoption path nor the preservation one.
const liveShapedSettings = `{
  "model": "opus",
  "hooks": {
    "SessionStart": [
      { "matcher": "", "hooks": [ { "type": "command", "command": "DOTF mem session-start", "timeout": 30 } ] },
      { "matcher": "", "hooks": [ { "type": "command", "command": "orca-session-start", "timeout": 5 } ] }
    ],
    "SessionEnd": [
      { "matcher": "", "hooks": [ { "type": "command", "command": "DOTF mem session-end", "timeout": 30 } ] }
    ],
    "PreToolUse": [
      { "matcher": "", "hooks": [ { "type": "command", "command": "orca-pre-tool", "timeout": 5 } ] }
    ],
    "Stop": [
      { "matcher": "", "hooks": [ { "type": "command", "command": "orca-stop", "timeout": 5 } ] }
    ]
  }
}`

func bindFixture(t *testing.T, settings string) (home, dotf string) {
	t.Helper()
	home = t.TempDir()
	dotf = filepath.Join(home, ".local", "bin", "dotf")
	if settings == "" {
		return home, dotf
	}
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := []byte(replaceAll(settings, "DOTF", dotf))
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), body, 0o600); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	return home, dotf
}

func replaceAll(s, old, new string) string {
	out := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			return out + s
		}
		out += s[:i] + new
		s = s[i+len(old):]
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func readSettings(t *testing.T, home string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("settings is not valid JSON after bind: %v", err)
	}
	return doc
}

func hookCommands(t *testing.T, doc map[string]any, event string) []string {
	t.Helper()
	hooks, _ := doc["hooks"].(map[string]any)
	groups, _ := hooks[event].([]any)
	var out []string
	for _, g := range groups {
		group, _ := g.(map[string]any)
		inner, _ := group["hooks"].([]any)
		for _, h := range inner {
			obj, _ := h.(map[string]any)
			cmd, _ := obj["command"].(string)
			out = append(out, cmd)
		}
	}
	return out
}

// TestBindAdoptsTheUnmarkedMemHookInsteadOfDuplicating is the defect this
// command was built around, from the adopting side.
//
// The deployed SessionStart hook runs our exact command and carries no marker.
// `isOurs` does not recognise it (its substring fallback only covers the gate),
// so without `sameCommand`'s exact-command adoption rule, bind would APPEND a
// second entry and every session would run `dotf mem session-start` twice.
func TestBindAdoptsTheUnmarkedMemHookInsteadOfDuplicating(t *testing.T) {
	home, dotf := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)

	if _, _, err := captureRealStreams(t, "harness", "bind",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", dotf); err != nil {
		t.Fatalf("bind: %v", err)
	}

	got := hookCommands(t, readSettings(t, home), "SessionStart")
	mem := 0
	for _, c := range got {
		if c == dotf+" mem session-start" {
			mem++
		}
	}
	if mem != 1 {
		t.Errorf("SessionStart carries %d copies of the mem hook, want exactly 1:\n%v", mem, got)
	}
}

// TestBindNeverTouchesAForeignHook is the defect from the destroying side.
//
// `merge_claude_settings` assigns `.hooks.SessionStart = $tmpl.hooks.SessionStart`.
// Simulated against a copy of the live file, that took SessionStart from two
// groups to one and deleted Orca's. This asserts the replacement does not: every
// foreign command present before bind is present after, and the count of foreign
// entries is unchanged.
func TestBindNeverTouchesAForeignHook(t *testing.T) {
	home, dotf := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)

	before := readSettings(t, home)
	beforeForeign := harness.ForeignHookCount(before)

	if _, _, err := captureRealStreams(t, "harness", "bind",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", dotf); err != nil {
		t.Fatalf("bind: %v", err)
	}
	after := readSettings(t, home)

	for _, ev := range []string{"SessionStart", "PreToolUse", "Stop"} {
		for _, want := range hookCommands(t, before, ev) {
			if want == dotf+" mem session-start" || want == dotf+" mem session-end" {
				continue // ours; adopted and re-emitted with a marker
			}
			if !contains(hookCommands(t, after, ev), want) {
				t.Errorf("bind deleted a foreign %s hook: %q\nafter: %v",
					ev, want, hookCommands(t, after, ev))
			}
		}
	}

	// Ours become marked, so the foreign count DROPS by exactly the two we
	// adopted -- never below that, which would mean a third party's entry was
	// claimed or removed.
	if got, want := harness.ForeignHookCount(after), beforeForeign-2; got != want {
		t.Errorf("foreign hook count = %d, want %d (was %d before)", got, want, beforeForeign)
	}
}

// TestBindAppendsTheGateBesideOrcasPreToolUseGroup pins AC5: a new hook on an
// event a third party already owns is a NEW group, not an edit of theirs.
func TestBindAppendsTheGateBesideOrcasPreToolUseGroup(t *testing.T) {
	home, dotf := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)

	if _, _, err := captureRealStreams(t, "harness", "bind",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", dotf); err != nil {
		t.Fatalf("bind: %v", err)
	}

	got := hookCommands(t, readSettings(t, home), "PreToolUse")
	if !contains(got, "orca-pre-tool") {
		t.Errorf("Orca's PreToolUse hook is gone: %v", got)
	}
	if !contains(got, dotf+" harness gate --harness claude") {
		t.Errorf("the gate was not emitted on PreToolUse: %v", got)
	}
}

// TestBindIsIdempotent is the doctrine's changed=0 on re-run, asserted on bytes
// rather than on the command's own report.
func TestBindIsIdempotent(t *testing.T) {
	home, dotf := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)
	args := []string{"harness", "bind", "--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", dotf}

	if _, _, err := captureRealStreams(t, args...); err != nil {
		t.Fatalf("first bind: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	stdout, _, err := captureRealStreams(t, args...)
	if err != nil {
		t.Fatalf("second bind: %v", err)
	}
	second, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Errorf("a re-run rewrote the file:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if !containsSub(stdout, "already current") {
		t.Errorf("a no-op run did not say so: %q", stdout)
	}
}

// TestBindSkipsWhatTheManifestSaysNotToEmit pins that emit:false is honoured and
// SAID OUT LOUD. A silent skip is how a gap stops being visible.
func TestBindSkipsWhatTheManifestSaysNotToEmit(t *testing.T) {
	home, dotf := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)

	stdout, _, err := captureRealStreams(t, "harness", "bind",
		"--repo-root", root, "--home", home, "--dotf-path", dotf)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	for _, agent := range []string{"pi", "opencode"} {
		if !containsSub(stdout, "skip "+agent) {
			t.Errorf("%s is declared emit:false and bind did not report skipping it:\n%s", agent, stdout)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".pi", "agent", "extensions", "dotfiles-gate.ts")); err == nil {
		t.Error("bind wrote a ts-extension it is declared not to emit")
	}
}

// TestBindRefusesToOverwriteUnparseableSettings: a file someone is mid-edit is
// not a file to bootstrap over.
func TestBindRefusesToOverwriteUnparseableSettings(t *testing.T) {
	home, dotf := bindFixture(t, `{"hooks": {`)
	root := repoRootForTest(t)

	_, _, err := captureRealStreams(t, "harness", "bind",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", dotf)
	if err == nil {
		t.Fatal("want an error on unparseable settings, got none")
	}
	if !containsSub(err.Error(), "refusing to overwrite") {
		t.Errorf("error does not say it refused: %v", err)
	}
	raw, readErr := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if readErr != nil || string(raw) != `{"hooks": {` {
		t.Errorf("the unparseable file was modified: %q (%v)", raw, readErr)
	}
}

// TestBindBootstrapsAnAbsentFile: nothing to preserve, so writing is safe.
func TestBindBootstrapsAnAbsentFile(t *testing.T) {
	home, dotf := bindFixture(t, "")
	root := repoRootForTest(t)

	if _, _, err := captureRealStreams(t, "harness", "bind",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", dotf); err != nil {
		t.Fatalf("bind: %v", err)
	}
	got := hookCommands(t, readSettings(t, home), "PreToolUse")
	if !contains(got, dotf+" harness gate --harness claude") {
		t.Errorf("bootstrap did not emit the gate: %v", got)
	}
}

// TestBindDryRunWritesNothing.
func TestBindDryRunWritesNothing(t *testing.T) {
	home, dotf := bindFixture(t, liveShapedSettings)
	root := repoRootForTest(t)
	path := filepath.Join(home, ".claude", "settings.json")

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	stdout, _, err := captureRealStreams(t, "harness", "bind", "--dry-run",
		"--harness", "claude", "--repo-root", root, "--home", home, "--dotf-path", dotf)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Error("--dry-run wrote to the settings file")
	}
	if !containsSub(stdout, "would update") {
		t.Errorf("--dry-run did not report the pending change: %q", stdout)
	}
}

// TestManifestGateEventMatchesDeclaredActionEvent pins the one duplication the
// manifest carries: `events.action` documents where the gate goes (measured per
// harness), and `emit_hooks` is what actually emits there. Two places saying the
// same thing drift; this makes the drift a failing test instead of a silent
// mismatch between the documentation and the emission.
func TestManifestGateEventMatchesDeclaredActionEvent(t *testing.T) {
	targets, err := harness.LoadBindTargets(repoRootForTest(t))
	if err != nil {
		t.Fatalf("load bind targets: %v", err)
	}
	for _, tgt := range targets {
		for _, h := range tgt.EmitHooks {
			if h.ID != "gate" {
				continue
			}
			if want := tgt.Events["action"]; h.Event != want {
				t.Errorf("%s: emit_hooks gate is on %q, events.action declares %q",
					tgt.Agent, h.Event, want)
			}
		}
	}
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

func containsSub(s, sub string) bool { return indexOf(s, sub) >= 0 }
