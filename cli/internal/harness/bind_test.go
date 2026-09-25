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

// Gemini CLI's groups (~/.gemini/settings.json) carry NO matcher key. Emitting
// claude's shape there would be assuming a schema from a family resemblance,
// which is how a hook came to be bound into that file for agy, a different
// product that never reads it.
func TestMergeHooksOmitsMatcherWhenTheHarnessHasNone(t *testing.T) {
	doc := decode(t, `{"hooks":{}}`)
	out, _, err := MergeHooks(doc, []HookCommand{{
		ID:      "gate",
		Event:   "BeforeTool",
		Command: "dotf harness gate --harness gemini --role reviewer",
	}})
	if err != nil {
		t.Fatal(err)
	}
	groups := out["hooks"].(map[string]any)["BeforeTool"].([]any)
	if _, has := groups[0].(map[string]any)["matcher"]; has {
		t.Error("a harness whose groups carry no matcher must not be given one")
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
	// agy's PreToolUse is in the set above; PreInvocation is the other event a
	// person waits on, since it runs before every model call.
	interactive["PreInvocation"] = true

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
			if target.Format == NamedHooksFormat {
				assertNamedHookCarriesItsTimeout(t, target.Agent, c)
				continue
			}
			assertHookCarriesItsTimeout(t, target.Agent, c)
		}
	}
	// C15: zero hooks checked is not a pass. It is the manifest having moved.
	if checked == 0 {
		t.Fatal("no interactive hook was checked — the manifest's emit_hooks moved, and this guard silently stopped guarding")
	}
}

// assertHookCarriesItsTimeout checks one hook THROUGH THE REAL EMISSION PATH.
//
// HookCommand's own field is not what a harness reads: bind.go writes `timeout`
// into the merged settings document and omits it when zero, so this is the only
// assertion that sees what actually reaches the file.
//
// IT COMPARES THE VALUE, not merely the key's presence. The first version
// asserted `strings.Contains(raw, "timeout")`, which the reviewer on #1471
// caught: that passes on a timeout of any value, including one that disagrees
// with the manifest, and it would also pass on the word appearing anywhere else
// in the document. A guard that is satisfied by a substring is the same class as
// the `grep -c` that counted subtests — green, and measuring the wrong thing.
func assertHookCarriesItsTimeout(t *testing.T, agent string, c HookCommand) {
	t.Helper()
	if c.Timeout <= 0 {
		t.Errorf("%s hook %q (%s) is unbounded: an interactive hook with no timeout stalls the user",
			agent, c.ID, c.Event)
		return
	}
	merged, _, err := MergeHooks(map[string]any{}, []HookCommand{c})
	if err != nil {
		t.Fatalf("MergeHooks for %q: %v", c.ID, err)
	}
	raw, err := json.Marshal(merged)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
				Timeout *int   `json:"timeout"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("emitted settings for %q are not the expected shape: %v (%s)", c.ID, err, raw)
	}
	found := false
	for _, group := range doc.Hooks[c.Event] {
		for _, h := range group.Hooks {
			if h.Command != c.Command {
				continue
			}
			found = true
			if h.Timeout == nil {
				t.Errorf("%s hook %q reaches the settings file with no timeout: %s", agent, c.ID, raw)
			} else if *h.Timeout != c.Timeout {
				t.Errorf("%s hook %q emits timeout %d, manifest declares %d", agent, c.ID, *h.Timeout, c.Timeout)
			}
		}
	}
	if !found {
		t.Errorf("%s hook %q never reached the emitted settings at all: %s", agent, c.ID, raw)
	}
}

// assertNamedHookCarriesItsTimeout is assertHookCarriesItsTimeout for a target
// whose file is a document of named hooks. It goes through MergeNamedHooks for
// the same reason: the emitted document, not the declaration, is what a harness
// reads, and its shape differs by event (grouped for the tool events, flat for
// the rest).
func assertNamedHookCarriesItsTimeout(t *testing.T, agent string, c HookCommand) {
	t.Helper()
	if c.Timeout <= 0 {
		t.Errorf("%s hook %q (%s) is unbounded: an interactive hook with no timeout stalls the user",
			agent, c.ID, c.Event)
		return
	}
	merged, _, err := MergeNamedHooks(map[string]any{}, BindMarker, []HookCommand{c})
	if err != nil {
		t.Fatalf("MergeNamedHooks for %q: %v", c.ID, err)
	}
	raw, err := json.Marshal(merged[BindMarker])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string][]struct {
		Command string `json:"command"`
		Timeout *int   `json:"timeout"`
		Hooks   []struct {
			Command string `json:"command"`
			Timeout *int   `json:"timeout"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("emitted hooks for %q are not the expected shape: %v (%s)", c.ID, err, raw)
	}
	found := false
	check := func(command string, timeout *int) {
		if command != c.Command {
			return
		}
		found = true
		if timeout == nil {
			t.Errorf("%s hook %q reaches hooks.json with no timeout: %s", agent, c.ID, raw)
		} else if *timeout != c.Timeout {
			t.Errorf("%s hook %q emits timeout %d, manifest declares %d", agent, c.ID, *timeout, c.Timeout)
		}
	}
	for _, entry := range doc[c.Event] {
		check(entry.Command, entry.Timeout)
		for _, h := range entry.Hooks {
			check(h.Command, h.Timeout)
		}
	}
	if !found {
		t.Errorf("%s hook %q never reached the emitted hooks at all: %s", agent, c.ID, raw)
	}
}

// A trimmed copy of ~/.gemini/config/hooks.json measured 2026-09-24. Orca's group
// is the only thing in it: five events, two grouped and three flat, which is the
// split agy's own documentation draws. It is agy's file, not Gemini CLI's
// settings.json, that agy reads its hooks from.
const deployedAgyHooks = `{
  "orca-status": {
    "PreInvocation":  [{"type":"command","command":"sh /o/antigravity-hook.sh","timeout":10}],
    "PostInvocation": [{"type":"command","command":"sh /o/antigravity-hook.sh","timeout":10}],
    "Stop":           [{"type":"command","command":"sh /o/antigravity-hook.sh","timeout":10}],
    "PreToolUse":     [{"matcher":"*","hooks":[{"type":"command","command":"sh /o/antigravity-hook.sh","timeout":10}]}],
    "PostToolUse":    [{"matcher":"*","hooks":[{"type":"command","command":"sh /o/antigravity-hook.sh","timeout":10}]}]
  }
}`

func agyGate() HookCommand {
	return HookCommand{
		Event: "PreToolUse", ID: "gate", UseMatcher: true, Timeout: 5,
		Command: "/home/x/.local/bin/dotf harness gate --harness agy",
	}
}

// The constraint the whole design turns on, restated for agy: its file is shared
// with Orca, so ours must appear beside its group and leave it byte for byte.
func TestMergeNamedHooksLeavesEveryForeignNamedHookUntouched(t *testing.T) {
	doc := decode(t, deployedAgyHooks)
	orcaBefore, _ := json.Marshal(doc["orca-status"])

	out, changed, err := MergeNamedHooks(doc, BindMarker, []HookCommand{agyGate()})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("adding our hook must report changed")
	}
	orcaAfter, _ := json.Marshal(out["orca-status"])
	if string(orcaBefore) != string(orcaAfter) {
		t.Errorf("Orca's group was altered:\nbefore %s\nafter  %s", orcaBefore, orcaAfter)
	}

	ours, _ := out[BindMarker].(map[string]any)
	groups, _ := ours["PreToolUse"].([]any)
	if len(groups) != 1 {
		t.Fatalf("want exactly one PreToolUse group under our name, got %v", ours)
	}
	group := groups[0].(map[string]any)
	if group["matcher"] != "*" {
		t.Errorf("a tool event's group needs a matcher; agy documents \"*\" as every tool, got %v", group["matcher"])
	}
	handler := group["hooks"].([]any)[0].(map[string]any)
	if handler["type"] != "command" || handler["command"] != agyGate().Command || handler["timeout"] != 5 {
		t.Errorf("handler = %v", handler)
	}
	// Ownership is by NAME: a sidecar field inside a handler is not known to be
	// tolerated by agy's decoder, and a rejected file would drop Orca's hooks too.
	raw, _ := json.Marshal(out[BindMarker])
	if strings.Contains(string(raw), managedKey) {
		t.Errorf("a named hook must not carry the %s sidecar: %s", managedKey, raw)
	}
}

func TestMergeNamedHooksIsIdempotentAndReplacesAChangedCommand(t *testing.T) {
	doc := decode(t, deployedAgyHooks)
	first, _, err := MergeNamedHooks(doc, BindMarker, []HookCommand{agyGate()})
	if err != nil {
		t.Fatal(err)
	}
	// Round-trip through JSON: a re-run reads the file back, where numbers are
	// float64 and maps are decoded fresh, and must still report no change.
	raw, _ := json.Marshal(first)
	second, changed, err := MergeNamedHooks(decode(t, string(raw)), BindMarker, []HookCommand{agyGate()})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("an unchanged run must report changed=false, so nothing is rewritten")
	}
	again, _ := json.Marshal(second)
	if string(raw) != string(again) {
		t.Errorf("a re-run altered the document:\n%s\n%s", raw, again)
	}

	moved := agyGate()
	moved.Command = "/opt/other/dotf harness gate --harness agy"
	third, changed, err := MergeNamedHooks(second, BindMarker, []HookCommand{moved})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("a changed command must report changed")
	}
	groups := third[BindMarker].(map[string]any)["PreToolUse"].([]any)
	if len(groups) != 1 {
		t.Errorf("a changed command must replace our entry, not accumulate a second: %v", groups)
	}
}

// agy documents two shapes and the wrong one is not an error, it is a hook that
// never fires: the tool events take a matcher group, the rest a flat list.
func TestMergeNamedHooksUsesTheShapeEachEventTakes(t *testing.T) {
	out, _, err := MergeNamedHooks(map[string]any{}, BindMarker, []HookCommand{
		{Event: "PreInvocation", ID: "a", Command: "x", Timeout: 3},
		{Event: "Stop", ID: "b", Command: "y"},
		{Event: "PostToolUse", ID: "c", Command: "z", Matcher: "run_command", UseMatcher: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	ours := out[BindMarker].(map[string]any)

	flat := ours["PreInvocation"].([]any)[0].(map[string]any)
	if flat["command"] != "x" || flat["type"] != "command" {
		t.Errorf("a flat event carries its handler directly, got %v", flat)
	}
	if _, wrapped := flat["hooks"]; wrapped {
		t.Error("a flat event must not wrap its handler in a group")
	}
	if _, has := ours["Stop"].([]any)[0].(map[string]any)["timeout"]; has {
		t.Error("a zero timeout is omitted so agy applies its own default")
	}

	group := ours["PostToolUse"].([]any)[0].(map[string]any)
	if group["matcher"] != "run_command" {
		t.Errorf("a declared matcher is kept, got %v", group["matcher"])
	}
}

// agy's /hooks command toggles `enabled`. Switching a hook back on that a person
// switched off would override an explicit choice.
func TestMergeNamedHooksKeepsAPersonsDisableSwitch(t *testing.T) {
	doc := decode(t, `{"dotfiles-harness":{"enabled":false,"PreToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"old"}]}]}}`)
	out, changed, err := MergeNamedHooks(doc, BindMarker, []HookCommand{agyGate()})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("the stale command must still be brought up to date")
	}
	ours := out[BindMarker].(map[string]any)
	if ours["enabled"] != false {
		t.Errorf("enabled was reset: %v", ours["enabled"])
	}
	fresh, _, _ := MergeNamedHooks(map[string]any{}, BindMarker, []HookCommand{agyGate()})
	if _, has := fresh[BindMarker].(map[string]any)["enabled"]; has {
		t.Error("a first emission must not invent an enabled field")
	}
}

func TestMergeNamedHooksRefusesWhatCannotBeEmitted(t *testing.T) {
	if _, _, err := MergeNamedHooks(map[string]any{}, "", []HookCommand{agyGate()}); err == nil {
		t.Error("a hook with no name would be unowned")
	}
	if _, _, err := MergeNamedHooks(map[string]any{}, BindMarker, []HookCommand{{Event: "PreToolUse"}}); err == nil {
		t.Error("a hook with no command must be refused, not emitted empty")
	}
}

// The gate moved from Gemini CLI's file to agy's. Retiring the old entry removes
// exactly ours and nothing beside it.
func TestRetireHooksRemovesOnlyOurEntry(t *testing.T) {
	doc := decode(t, `{"hooks":{
	  "BeforeTool":[
	    {"hooks":[{"type":"command","command":"sh /o/gemini-hook.sh"}]},
	    {"hooks":[{"_managed":"dotfiles-harness:gate","type":"command","command":"dotf harness gate --harness agy"}]},
	    {"hooks":[{"type":"command","command":"sh /o/mixed.sh"},{"_managed":"dotfiles-harness:gate","type":"command","command":"dotf harness gate --harness agy"}]}
	  ],
	  "AfterTool":[{"hooks":[{"type":"command","command":"sh /o/gemini-hook.sh"}]}]
	}}`)
	foreignBefore := ForeignHookCount(doc)

	out, changed := RetireHooks(doc, "BeforeTool", "gate")
	if !changed {
		t.Fatal("our entries were present, so this must report changed")
	}
	if got := ForeignHookCount(out); got != foreignBefore {
		t.Errorf("foreign hooks went from %d to %d", foreignBefore, got)
	}
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "harness gate") {
		t.Errorf("one of our entries survived: %s", raw)
	}
	groups := out["hooks"].(map[string]any)["BeforeTool"].([]any)
	if len(groups) != 2 {
		t.Errorf("the group that held only ours goes; the mixed one stays with its foreign hook: %v", groups)
	}
	if _, again := RetireHooks(out, "BeforeTool", "gate"); again {
		t.Error("a second retirement must find nothing to do")
	}
}

// An event with nothing left goes with it, so the file does not accumulate empty
// arrays, and an event we never touched is left exactly as it was.
func TestRetireHooksDropsAnEventItEmptied(t *testing.T) {
	doc := decode(t, `{"hooks":{
	  "BeforeTool":[{"hooks":[{"_managed":"dotfiles-harness:gate","type":"command","command":"x"}]}],
	  "AfterTool":[{"hooks":[{"type":"command","command":"keep"}]}]
	}}`)
	out, changed := RetireHooks(doc, "BeforeTool", "gate")
	if !changed {
		t.Fatal("expected a change")
	}
	hooks := out["hooks"].(map[string]any)
	if _, has := hooks["BeforeTool"]; has {
		t.Error("an emptied event must be removed")
	}
	if _, has := hooks["AfterTool"]; !has {
		t.Error("an event we did not touch was removed")
	}
	if _, changed := RetireHooks(decode(t, `{"other":1}`), "BeforeTool", "gate"); changed {
		t.Error("a document with no hooks key has nothing to retire")
	}
}

// TestBindKeyHoldsOnlyFormatsAnOlderBinaryUnderstands is the guard for a skew that
// was reproduced, not imagined.
//
// The manifest is mirrored by setup as soon as it merges; the binary is released
// separately. A binary from before the hooks-json format existed reads
// `agents.bind`, ignores every key it does not know, and treats every target in it
// as claude's shape. dotf 0.57.0 given a manifest with the agy target in `bind`
// wrote a top-level `hooks` key, with a `_managed` sidecar inside a handler, into a
// copy of the real ~/.gemini/config/hooks.json: a file agy decodes as protojson and
// shares with Orca.
//
// So a format an older binary does not know must live where it never looks. This
// asserts that on the REAL manifest, by consequence: every format found in `bind`
// is one every released binary handles, and the agy target is present under
// `bind_named`, so the guard cannot pass on a manifest that lost the target.
func TestBindKeyHoldsOnlyFormatsAnOlderBinaryUnderstands(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ManifestFile))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m struct {
		Agents struct {
			Bind []struct {
				Agent  string `json:"agent"`
				File   string `json:"file"`
				Format string `json:"format"`
			} `json:"bind"`
			BindNamed []struct {
				Agent  string `json:"agent"`
				Format string `json:"format"`
			} `json:"bind_named"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	// The formats dotf could emit BEFORE hooks-json existed.
	understoodByOlderBinaries := map[string]bool{"command-hook": true, "ts-extension": true}
	if len(m.Agents.Bind) == 0 {
		t.Fatal("no agents.bind targets were checked; the manifest moved and this guard silently stopped guarding")
	}
	for _, b := range m.Agents.Bind {
		if !understoodByOlderBinaries[b.Format] {
			t.Errorf("agents.bind target %q has format %q, which an older binary would emit as claude's shape into %s; "+
				"declare it under agents.bind_named", b.Agent, b.Format, b.File)
		}
	}

	found := false
	for _, b := range m.Agents.BindNamed {
		if b.Agent == "agy" && b.Format == NamedHooksFormat {
			found = true
		}
	}
	if !found {
		t.Error("the agy target is not under agents.bind_named; either it was lost or it moved to a key an older binary reads")
	}
}

// LoadBindTargets refuses a target in the wrong key when the manifest loads, so a
// misplaced entry cannot reach an emitter at all.
func TestLoadBindTargetsRefusesAFormatInTheWrongKey(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "harness"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ManifestFile), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return root
	}
	claude := `{"agent":"claude","file":".claude/settings.json","format":"command-hook"}`
	agy := `{"agent":"agy","file":".gemini/config/hooks.json","format":"hooks-json"}`

	t.Run("both keys are read and merged", func(t *testing.T) {
		root := write(t, `{"agents":{"bind":[`+claude+`],"bind_named":[`+agy+`]}}`)
		got, err := LoadBindTargets(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0].Agent != "claude" || got[1].Agent != "agy" {
			t.Errorf("targets = %+v, want claude then agy", got)
		}
	})
	t.Run("a hooks-json target under bind is refused", func(t *testing.T) {
		root := write(t, `{"agents":{"bind":[`+claude+`,`+agy+`]}}`)
		if _, err := LoadBindTargets(root); err == nil || !strings.Contains(err.Error(), "bind_named") {
			t.Errorf("want a refusal naming bind_named, got %v", err)
		}
	})
	t.Run("another format under bind_named is refused", func(t *testing.T) {
		root := write(t, `{"agents":{"bind":[`+claude+`],"bind_named":[{"agent":"x","file":"f","format":"command-hook"}]}}`)
		if _, err := LoadBindTargets(root); err == nil {
			t.Error("a claude-shaped target under bind_named must be refused")
		}
	})
	t.Run("an empty bind is still an error", func(t *testing.T) {
		root := write(t, `{"agents":{"bind_named":[`+agy+`]}}`)
		if _, err := LoadBindTargets(root); err == nil {
			t.Error("no bind targets must stay an error: it is indistinguishable from a manifest that moved")
		}
	})
}
