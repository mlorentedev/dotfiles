package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var d map[string]any
	if err := json.Unmarshal([]byte(s), &d); err != nil {
		t.Fatal(err)
	}
	return d
}

// A trimmed copy of the shape measured on this machine 2026-08-26: our
// SessionStart, plus foreign entries Orca owns — including PreToolUse, the very
// event the gate needs.
const deployedClaudeSettings = `{
  "hooks": {
    "SessionStart": [{"matcher":"","hooks":[{"type":"command","command":"/home/x/.local/bin/dotf mem session-start","timeout":30}]}],
    "PreToolUse":   [{"matcher":"*","hooks":[{"type":"command","command":"sh /home/x/.orca/agent-hooks/claude-hook.sh"}]}],
    "Stop":         [{"hooks":[{"type":"command","command":"sh /home/x/.orca/agent-hooks/claude-hook.sh"}]}],
    "PostToolUse":  [{"matcher":"*","hooks":[{"type":"command","command":"sh /home/x/.orca/agent-hooks/claude-hook.sh"}]}]
  }
}`

func gateCmd(role string) HookCommand {
	return HookCommand{
		Event:      "PreToolUse",
		ID:         "gate",
		Command:    "dotf harness gate --harness claude --role " + role,
		Matcher:    "*",
		UseMatcher: true,
	}
}

// AC6 — the constraint the whole design turns on. Orca owns 10 of 12 hook events
// on the measured machine, including PreToolUse. Emission must append beside it.
func TestMergeHooksPreservesForeignEntries(t *testing.T) {
	doc := decode(t, deployedClaudeSettings)
	before := ForeignHookCount(doc)
	if before == 0 {
		t.Fatal("the fixture must contain foreign hooks or this test proves nothing")
	}

	out, changed, err := MergeHooks(doc, []HookCommand{gateCmd("reviewer")})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("adding a hook must report changed")
	}
	if after := ForeignHookCount(out); after != before {
		t.Errorf("foreign hooks went from %d to %d — emission clobbered a third party", before, after)
	}

	// Orca's PreToolUse entry specifically, since that is the contested event.
	raw, _ := json.Marshal(out)
	if !strings.Contains(string(raw), "orca-hooks/claude-hook.sh") && !strings.Contains(string(raw), "agent-hooks/claude-hook.sh") {
		t.Error("Orca's PreToolUse hook did not survive the merge")
	}
	if !strings.Contains(string(raw), "dotf harness gate") {
		t.Error("our hook was not emitted")
	}
}

// AC5 — idempotent under RE-RUN.
func TestMergeHooksIsIdempotent(t *testing.T) {
	doc := decode(t, deployedClaudeSettings)
	out, _, err := MergeHooks(doc, []HookCommand{gateCmd("reviewer")})
	if err != nil {
		t.Fatal(err)
	}
	again, changed, err := MergeHooks(out, []HookCommand{gateCmd("reviewer")})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("a second identical merge must report changed=false")
	}
	if got := countOurs(again, "PreToolUse"); got != 1 {
		t.Errorf("re-running produced %d of our entries, want exactly 1", got)
	}
}

// The third assertion AC5 and AC6 imply but neither states: idempotence under
// CHANGE. A changed command must REPLACE ours, not accumulate a second — which
// is the case that actually happens, since re-emission exists precisely because
// the command changed.
func TestMergeHooksReplacesOurOwnEntryRatherThanAccumulating(t *testing.T) {
	doc := decode(t, deployedClaudeSettings)
	out, _, err := MergeHooks(doc, []HookCommand{gateCmd("reviewer")})
	if err != nil {
		t.Fatal(err)
	}
	out, changed, err := MergeHooks(out, []HookCommand{gateCmd("builder")})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("a changed command must report changed")
	}
	if got := countOurs(out, "PreToolUse"); got != 1 {
		t.Fatalf("a changed command produced %d of our entries, want 1", got)
	}
	raw, _ := json.Marshal(out)
	if !strings.Contains(string(raw), "--role builder") || strings.Contains(string(raw), "--role reviewer") {
		t.Error("the old command was not replaced")
	}
}

// The latent bug this design replaces: `merge_claude_settings` writes
// `.hooks.<event>[0].hooks[0].command`, so a foreign group sitting at index 0
// gets silently overwritten. Find-by-marker must survive that ordering.
func TestMergeHooksSurvivesAForeignGroupAtIndexZero(t *testing.T) {
	doc := decode(t, `{"hooks":{"PreToolUse":[
	  {"matcher":"*","hooks":[{"type":"command","command":"sh /third/party/first.sh"}]}
	]}}`)
	out, _, err := MergeHooks(doc, []HookCommand{gateCmd("reviewer")})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(out)
	if !strings.Contains(string(raw), "/third/party/first.sh") {
		t.Fatal("a foreign hook at index 0 was overwritten — the positional bug survived")
	}
	if got := countOurs(out, "PreToolUse"); got != 1 {
		t.Errorf("want exactly 1 of ours alongside it, got %d", got)
	}
}

// agy's groups carry NO matcher key (measured against ~/.gemini/settings.json).
// Emitting claude's shape there would be assuming a schema from a family
// resemblance.
func TestMergeHooksOmitsMatcherWhenTheHarnessHasNone(t *testing.T) {
	doc := decode(t, `{"hooks":{}}`)
	out, _, err := MergeHooks(doc, []HookCommand{{
		ID:      "gate",
		Event:   "BeforeTool",
		Command: "dotf harness gate --harness agy --role reviewer",
	}})
	if err != nil {
		t.Fatal(err)
	}
	groups := out["hooks"].(map[string]any)["BeforeTool"].([]any)
	if _, has := groups[0].(map[string]any)["matcher"]; has {
		t.Error("agy's group must not carry a matcher key")
	}
}

// An entry written before the marker existed must still be recognised, or every
// run appends a duplicate.
func TestMergeHooksRecognisesAPreMarkerEntry(t *testing.T) {
	doc := decode(t, `{"hooks":{"PreToolUse":[
	  {"matcher":"*","hooks":[{"type":"command","command":"dotf harness gate --harness claude --role old"}]}
	]}}`)
	out, changed, err := MergeHooks(doc, []HookCommand{gateCmd("reviewer")})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("an outdated pre-marker entry must be updated")
	}
	if got := countOurs(out, "PreToolUse"); got != 1 {
		t.Errorf("want 1 entry after adopting a pre-marker one, got %d", got)
	}
}

// A group shape the merge does not understand is left exactly as it is:
// normalising a foreign entry is the one thing this must never do.
func TestMergeHooksLeavesUnknownGroupShapesAlone(t *testing.T) {
	doc := decode(t, `{"hooks":{"PreToolUse":["a bare string someone put here"]}}`)
	out, _, err := MergeHooks(doc, []HookCommand{gateCmd("reviewer")})
	if err != nil {
		t.Fatal(err)
	}
	groups := out["hooks"].(map[string]any)["PreToolUse"].([]any)
	if s, ok := groups[0].(string); !ok || s != "a bare string someone put here" {
		t.Errorf("an unrecognised group was rewritten: %#v", groups[0])
	}
}

// Adoption, not duplication: an unmarked entry already running exactly our
// command is ours. Measured need — the deployed SessionStart runs
// `dotf mem session-start`, written by the positional path, carrying no marker
// and matching no substring, so without adoption `bind` would append a second
// identical hook on its first run.
func TestMergeHooksAdoptsAnUnmarkedEntryRunningOurExactCommand(t *testing.T) {
	cmd := "/home/x/.local/bin/dotf mem session-start"
	doc := decode(t, `{"hooks":{"SessionStart":[
	  {"matcher":"","hooks":[{"type":"command","command":"`+cmd+`","timeout":30}]}
	]}}`)
	out, _, err := MergeHooks(doc, []HookCommand{{
		ID: "mem", Event: "SessionStart", Command: cmd, Matcher: "", UseMatcher: true, Timeout: 30,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := countOurs(out, "SessionStart"); got != 1 {
		t.Fatalf("adoption produced %d entries, want 1 — a duplicate hook would run twice per event", got)
	}
	// And it must NOT adopt a third party's different command on the same event.
	out, _, err = MergeHooks(out, []HookCommand{{
		ID: "gate", Event: "SessionStart", Command: "dotf harness gate --harness claude --role reviewer",
		Matcher: "", UseMatcher: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(out)
	if !strings.Contains(string(raw), "mem session-start") {
		t.Error("adopting one entry must not have replaced an unrelated command")
	}
}

func TestMergeHooksRejectsAnEmptyCommand(t *testing.T) {
	if _, _, err := MergeHooks(map[string]any{}, []HookCommand{{ID: "gate", Event: "PreToolUse"}}); err == nil {
		t.Error("an empty command must be an error, not an emitted no-op hook")
	}
}

// Effect on real data: the actual deployed settings file, if present.
func TestMergeAgainstTheRealDeployedSettings(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("no HOME")
	}
	raw, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Skip("no deployed claude settings on this machine")
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Skipf("deployed settings unreadable: %v", err)
	}
	before := foreignCommands(doc)
	out, _, err := MergeHooks(doc, []HookCommand{gateCmd("reviewer")})
	if err != nil {
		t.Fatalf("merging into the real file failed: %v", err)
	}
	after := foreignCommands(out)

	// A COUNT is the wrong assertion here, and it fails on any machine bound
	// before the marker field existed. Such a machine carries an UNMARKED gate
	// entry, which `ForeignHookCount` counts as foreign and which `isOurs`
	// deliberately adopts by command substring — so the count drops by one while
	// nothing was lost. Measured on this box, 15 -> 14, with no third-party hook
	// touched.
	//
	// What the file actually has to promise is narrower and is the thing the
	// Orca incident was about: no hook BELONGING TO SOMEBODY ELSE disappears.
	// So the loss is compared per command, and the only tolerated disappearance
	// is an entry the merge was entitled to claim.
	for cmd, n := range before {
		lost := n - after[cmd]
		if lost <= 0 {
			continue
		}
		if strings.Contains(cmd, "dotf harness gate") {
			// A pre-marker entry of our own, adopted. This is the fallback
			// working, not a hook being deleted.
			continue
		}
		t.Errorf("a hook that is not ours was lost from the REAL file (%d of %d): %.120s", lost, n, cmd)
	}
	t.Logf("real deployed file: %d distinct foreign hook commands, none belonging to a third party lost", len(before))
}

func countOurs(doc map[string]any, event string) int {
	hooks, _ := doc["hooks"].(map[string]any)
	groups, _ := hooks[event].([]any)
	n := 0
	for _, g := range groups {
		group, ok := g.(map[string]any)
		if !ok {
			continue
		}
		inner, _ := group["hooks"].([]any)
		for _, h := range inner {
			if obj, ok := h.(map[string]any); ok && (isOurs(obj, "gate") || isOurs(obj, "mem")) {
				n++
			}
		}
	}
	return n
}

// foreignCommands is ForeignHookCount broken out per command string.
//
// The count alone cannot answer the question the test above asks. "Fifteen
// became fourteen" is true both when a third party's hook was deleted and when
// one of our own pre-marker entries was adopted, and those are opposite
// outcomes — one is the incident that motivated the marker, the other is the
// mechanism that prevents it.
func foreignCommands(doc map[string]any) map[string]int {
	out := map[string]int{}
	hooks, _ := doc["hooks"].(map[string]any)
	for _, v := range hooks {
		groups, _ := v.([]any)
		for _, g := range groups {
			group, ok := g.(map[string]any)
			if !ok {
				continue
			}
			inner, _ := group["hooks"].([]any)
			for _, h := range inner {
				obj, ok := h.(map[string]any)
				if !ok {
					continue
				}
				if m, ok := obj[managedKey].(string); ok && strings.HasPrefix(m, BindMarker+":") {
					continue
				}
				cmd, _ := obj["command"].(string)
				out[cmd]++
			}
		}
	}
	return out
}

// TestEveryInteractiveHookIsBounded closes HARNESS-109's AC7.
//
// THE GATE HOOK SHIPPED UNBOUNDED. `harness suggest --from-hook` was given
// `timeout: 5` by #1455's review, because an unbounded hook on the interactive
// path delays what the user typed; `harness gate` runs on EVERY tool call on the
// same path and carried no timeout at all. The asymmetry was invisible because
// nothing asserted the class — only the one hook the review happened to read.
//
// Measured in the Claude Code 2.1.260 executable: a timed-out hook returns
// `blocked: false` unless `timeoutFailsClosed` is set, and that flag is only set
// for a call served to a cloud session. So locally a bound converts a long stall
// into a fast fail-open — which is the gate's own documented contract — and
// cannot turn into a refused tool call. Bounding is strictly safer, never a new
// blocking risk.
//
// Asserted BY CONSEQUENCE on the rendered hook commands, not by string-matching
// the manifest: bind.go omits `timeout` from the emitted JSON when it is zero,
// so a declaration test would pass on a value that never reaches the harness.
func TestEveryInteractiveHookIsBounded(t *testing.T) {
	targets, err := LoadBindTargets(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("LoadBindTargets: %v", err)
	}

	// The events a human is waiting on. A session-lifecycle hook may legitimately
	// take longer, and both already carry 30.
	interactive := map[string]bool{
		"PreToolUse": true, "PostToolUse": true,
		"UserPromptSubmit": true, "BeforeTool": true,
	}

	checked := 0
	for _, target := range targets {
		if !target.Emits() {
			continue
		}
		cmds, err := target.HookCommands("/opt/bin/dotf")
		if err != nil {
			t.Fatalf("HookCommands for %q: %v", target.Agent, err)
		}
		for _, c := range cmds {
			if !interactive[c.Event] {
				continue
			}
			checked++
			if c.Timeout <= 0 {
				t.Errorf("%s hook %q (%s) is unbounded: an interactive hook with no timeout stalls the user",
					target.Agent, c.ID, c.Event)
			}
			// Through the REAL emission path. HookCommand's own field is not
			// what a harness reads: bind.go writes `timeout` into the merged
			// settings document and omits it when zero, so this is the only
			// assertion that sees what actually reaches the file.
			merged, _, err := MergeHooks(map[string]any{}, []HookCommand{c})
			if err != nil {
				t.Fatalf("MergeHooks for %q: %v", c.ID, err)
			}
			raw, err := json.Marshal(merged)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !strings.Contains(string(raw), `"timeout"`) {
				t.Errorf("%s hook %q reaches the settings file with no timeout: %s", target.Agent, c.ID, raw)
			}
		}
	}
	// C15: zero hooks checked is not a pass. It is the manifest having moved.
	if checked == 0 {
		t.Fatal("no interactive hook was checked — the manifest's emit_hooks moved, and this guard silently stopped guarding")
	}
}
