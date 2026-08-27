package harness

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// BindMarker identifies a hook entry this repository owns.
//
// OWNERSHIP IS BY MARKER, NEVER BY POSITION, and that replaces a latent bug
// rather than merely being tidier. `merge_claude_settings()` in setup-linux.sh
// writes `.hooks.SessionStart[0].hooks[0].command` — a positional claim that
// holds only because ours happens to sit at index 0 today. Measured 2026-08-26,
// the deployed ~/.claude/settings.json carries 12 events of which **10 belong to
// Orca**, and all four of agy's belong to Orca. The day a third party prepends a
// group to an event we also write, a positional writer silently overwrites a
// foreign hook.
const BindMarker = "dotfiles-harness"

// managedKey is the sidecar field marking our entry. It sits inside the hook
// object rather than the group so an event carrying several groups stays
// unambiguous.
const managedKey = "_managed"

// HookCommand is one emitted hook.
type HookCommand struct {
	// Event is the harness's event name (PreToolUse, BeforeTool, ...).
	Event string
	// Command is the shell line the harness runs. It calls `dotf harness gate`;
	// nothing else belongs here, because logic in a hook is logic that cannot be
	// tested without the harness.
	Command string
	// Matcher is emitted only when the harness's schema carries one. Claude's
	// groups have `matcher`; agy's — measured against ~/.gemini/settings.json —
	// do not. Emitting claude's shape into agy would be assuming a schema from a
	// family resemblance.
	Matcher string
	// UseMatcher distinguishes "matcher is empty" from "this harness has no
	// matcher key at all".
	UseMatcher bool
	// ID names THIS hook's purpose ("gate", "mem"). Identity is per-ID, not
	// per-repository, and that distinction was found by test rather than by
	// design: after adopting `dotf mem session-start` on SessionStart, emitting
	// the gate on the same event MATCHED the memory hook's marker and replaced
	// it, deleting a live hook. A repository that emits two different hooks on
	// one event needs per-purpose identity or the second silently evicts the
	// first.
	ID string
	// Timeout in seconds; omitted when zero.
	Timeout int
}

// MergeHooks folds our hooks into an existing settings document, returning the
// new document and whether anything changed.
//
// THE ALGORITHM IS FIND-BY-MARKER:
//
//   - ours, present   -> replaced in place (idempotence under CHANGE, not just
//     under re-run: a changed command must not accumulate a second entry)
//   - ours, absent    -> appended as a NEW group; no existing group is touched
//   - foreign entries -> never reordered, rewritten, or removed
//
// AC5 and AC6 both fall out of that, and so does the third assertion the pair
// implies but neither states: ours-then-updated.
//
// It takes and returns a decoded document rather than bytes so the caller owns
// formatting; a settings file is edited by humans and re-indenting the whole
// thing on every setup run would be a diff nobody asked for.
func MergeHooks(doc map[string]any, cmds []HookCommand) (map[string]any, bool, error) {
	if doc == nil {
		doc = map[string]any{}
	}
	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}

	changed := false
	// Sorted so a re-run produces byte-identical output; map order would make
	// the idempotence assertion flap.
	sorted := append([]HookCommand(nil), cmds...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Event < sorted[j].Event })

	for _, c := range sorted {
		if c.Event == "" || c.Command == "" {
			return nil, false, fmt.Errorf("hook for event %q has no command", c.Event)
		}
		if c.ID == "" {
			return nil, false, fmt.Errorf("hook for event %q has no ID — identity is per-purpose, or a second hook on the same event evicts the first", c.Event)
		}
		groups, _ := hooks[c.Event].([]any)
		updated, groupChanged, err := mergeEvent(groups, c)
		if err != nil {
			return nil, false, err
		}
		if groupChanged {
			changed = true
		}
		hooks[c.Event] = updated
	}

	doc["hooks"] = hooks
	return doc, changed, nil
}

// mergeEvent handles one event's group array.
func mergeEvent(groups []any, c HookCommand) ([]any, bool, error) {
	want := hookObject(c)

	for gi, g := range groups {
		group, ok := g.(map[string]any)
		if !ok {
			// A group shape we do not understand is left exactly as it is. The
			// alternative — normalising it — rewrites a foreign entry, which is
			// the one thing this merge must never do.
			continue
		}
		inner, _ := group["hooks"].([]any)
		for hi, h := range inner {
			obj, ok := h.(map[string]any)
			if !ok || (!isOurs(obj, c.ID) && !sameCommand(obj, c.Command)) {
				continue
			}
			if sameHook(obj, want) {
				return groups, false, nil
			}
			inner[hi] = want
			group["hooks"] = inner
			groups[gi] = group
			return groups, true, nil
		}
	}

	group := map[string]any{"hooks": []any{want}}
	if c.UseMatcher {
		group["matcher"] = c.Matcher
	}
	return append(groups, group), true, nil
}

// hookObject builds our hook entry, carrying the marker.
func hookObject(c HookCommand) map[string]any {
	obj := map[string]any{
		"type":     "command",
		"command":  c.Command,
		managedKey: BindMarker + ":" + c.ID,
	}
	if c.Timeout > 0 {
		obj["timeout"] = c.Timeout
	}
	return obj
}

// isOurs recognises our entry.
//
// The sidecar field is checked FIRST because it survives a command edit — the
// case that matters, since the whole point of re-emission is that the command
// changes. The command-substring fallback exists for entries written before the
// marker existed, and for a harness that strips unknown keys from a hook object:
// if one does, the sidecar silently vanishes and only the fallback would
// recognise our own entry, so losing it would make every run append a duplicate.
func isOurs(obj map[string]any, id string) bool {
	if m, ok := obj[managedKey].(string); ok && m == BindMarker+":"+id {
		return true
	}
	// Pre-marker gate entries, from before this field existed.
	if id == "gate" {
		cmd, _ := obj["command"].(string)
		return strings.Contains(cmd, "dotf harness gate")
	}
	return false
}

// sameCommand adopts an unmarked entry that already runs exactly the command we
// are about to emit.
//
// Without it, `bind` taking over a hook that `merge_claude_settings` wrote
// positionally would APPEND A DUPLICATE rather than adopt it: the existing entry
// carries no marker, and `isOurs`'s substring fallback only recognises the gate.
// Measured on the deployed file — `SessionStart` runs `dotf mem session-start`,
// which matches neither. Exact-command equality is the safe adoption rule: it
// cannot claim a third party's hook, because a hook running our exact command IS
// ours regardless of who wrote it.
func sameCommand(obj map[string]any, want string) bool {
	cmd, _ := obj["command"].(string)
	return want != "" && cmd == want
}

// sameHook reports whether the existing entry already says what we want, so an
// unchanged run reports changed=false rather than rewriting identical bytes.
func sameHook(got, want map[string]any) bool {
	a, err1 := json.Marshal(got)
	b, err2 := json.Marshal(want)
	return err1 == nil && err2 == nil && string(a) == string(b)
}

// ForeignHookCount reports how many hook entries in a document belong to
// somebody else. Used by the verification to assert AC6 on real data rather than
// on a fixture built to pass.
func ForeignHookCount(doc map[string]any) int {
	hooks, _ := doc["hooks"].(map[string]any)
	n := 0
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
				if m, ok := obj[managedKey].(string); !ok || !strings.HasPrefix(m, BindMarker+":") {
					n++
				}
			}
		}
	}
	return n
}
