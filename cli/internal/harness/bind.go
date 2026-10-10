package harness

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// BindMarker identifies a hook entry this repository owns.
//
// OWNERSHIP IS BY MARKER WHEN IT IS PRESENT, ELSE BY COMMAND SIGNATURE, NEVER BY
// POSITION. The marker is a key the harness does not know, and a co-owner of the
// file may drop it: measured 2026-10-10 (#2232), ~/.claude/settings.json was
// rewritten with every hook intact and every marker gone. So the marker is
// written but never required; see isOurs.
//
// Position-free identity replaces a latent bug rather than merely being tidier.
// `merge_claude_settings()` in setup-linux.sh wrote
// `.hooks.SessionStart[0].hooks[0].command` — a positional claim that held only
// because ours happened to sit at index 0. Measured 2026-08-26,
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
	// Event is the harness's event name (PreToolUse, PreInvocation, BeforeTool, ...).
	Event string
	// Command is the shell line the harness runs. It calls `dotf harness gate`;
	// nothing else belongs here, because logic in a hook is logic that cannot be
	// tested without the harness.
	Command string
	// Matcher is emitted only when the harness's schema carries one. Claude's
	// groups have `matcher`, and so do agy's hooks.json groups for the two tool
	// events; Gemini CLI's settings.json groups do not. Emitting one harness's
	// shape into another is assuming a schema from a family resemblance, which is
	// how the agy binding came to be written against Gemini CLI's file.
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
			if !ok || !isOurs(obj, c.ID, c.Command) {
				continue
			}
			if sameHook(withoutMarker(obj), withoutMarker(want)) {
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

// isOurs recognises our entry, in three ways, any of which is enough.
//
//   - The marker, when it is there: it survives a change to the command's
//     arguments, the case re-emission exists for.
//   - The command signature: a `dotf` binary followed by exactly the arguments
//     we emit, wherever the binary lives. This is what holds when a co-owner of
//     the file has dropped the marker (#2232), and it adopts an unmarked entry
//     that the positional setup path wrote. It cannot claim a third party's
//     hook: one running our binary with our arguments IS ours.
//   - Pre-marker gate entries, from before the marker existed, whose
//     arguments may differ from today's.
//
// What none of them recognises is an unmarked entry whose arguments changed
// since it was written, other than the gate: the manifest's retire list names
// such a hook by id, and only the marker or the gate rule finds it.
func isOurs(obj map[string]any, id, command string) bool {
	if m, ok := obj[managedKey].(string); ok && m == BindMarker+":"+id {
		return true
	}
	cmd, _ := obj["command"].(string)
	if got, ok := dotfArgs(cmd); ok {
		if want, ok := dotfArgs(command); ok && got == want {
			return true
		}
	}
	return id == "gate" && strings.Contains(cmd, "dotf harness gate")
}

// dotfArgs splits a hook command into the arguments after its binary, when the
// binary is dotf: bare or double-quoted (Windows quotes it), with either path
// separator, as `dotf` or `dotf.exe`.
func dotfArgs(cmd string) (string, bool) {
	var bin, rest string
	if strings.HasPrefix(cmd, `"`) {
		end := strings.Index(cmd[1:], `"`)
		if end < 0 {
			return "", false
		}
		bin, rest = cmd[1:end+1], cmd[end+2:]
	} else {
		bin, rest, _ = strings.Cut(cmd, " ")
	}
	base := bin[strings.LastIndexAny(bin, `/\`)+1:]
	if base != "dotf" && base != "dotf.exe" {
		return "", false
	}
	return strings.TrimLeft(rest, " "), true
}

// withoutMarker returns a hook object without the marker, so that whether an
// entry is current does not depend on a key a co-owner of the file may drop.
// Comparing with it, an entry that lost only its marker is current: rewriting
// it would put the marker back for the next writer to strip, and every run
// would report a change.
func withoutMarker(obj map[string]any) map[string]any {
	out := make(map[string]any, len(obj))
	for k, v := range obj {
		if k != managedKey {
			out[k] = v
		}
	}
	return out
}

// sameHook reports whether the existing entry already says what we want, so an
// unchanged run reports changed=false rather than rewriting identical bytes.
func sameHook(got, want map[string]any) bool {
	a, err1 := json.Marshal(got)
	b, err2 := json.Marshal(want)
	return err1 == nil && err2 == nil && string(a) == string(b)
}

// ForeignHookCount reports how many hook entries in a document belong to
// somebody else: neither marked ours nor running the dotf binary. Used by the
// verification to assert AC6 on real data rather than on a fixture built to
// pass. It is coarser than isOurs, which also requires the manifest's exact
// arguments: an unmarked dotf entry with other arguments is not counted here,
// and TestMergeHooksLeavesADotfEntryWithOtherArgumentsUntouched pins that bind
// still leaves it alone.
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
				m, _ := obj[managedKey].(string)
				cmd, _ := obj["command"].(string)
				if _, isDotf := dotfArgs(cmd); !isDotf && !strings.HasPrefix(m, BindMarker+":") {
					n++
				}
			}
		}
	}
	return n
}

// NamedHooksFormat is the `format` of a target whose file is a document of NAMED
// hooks - agy's hooks.json - rather than claude's `hooks` key of events.
const NamedHooksFormat = "hooks-json"

// namedGroupedEvents are the hooks.json events whose handlers sit inside a group
// carrying a `matcher`. Every other event takes its handlers as a flat list.
// agy's documentation draws the line here: "Grouped (uses matcher & hooks
// wrapper)" for PreToolUse and PostToolUse, "Flat (list of handler objects
// directly)" for PreInvocation, PostInvocation and Stop.
var namedGroupedEvents = map[string]bool{"PreToolUse": true, "PostToolUse": true}

// MergeNamedHooks writes our hooks into a document of named hooks, returning the
// new document and whether anything changed.
//
// THE SHAPE IS NOT claude's, so MergeHooks cannot serve it: the top-level keys
// are hook NAMES, each mapping to an object of events, and there is no `hooks`
// key at all. The file is shared - Orca owns `orca-status` in the same document -
// and agy merges every named hook for an event and runs them in turn, so ours
// coexists as a sibling key.
//
// OWNERSHIP IS BY NAME, which is what the format offers. There is deliberately no
// `_managed` sidecar inside a handler: agy decodes these files as protojson, an
// unknown field in a handler is not known to be tolerated, and a rejected
// hooks.json would silently drop every hook in it, Orca's included. We own the
// whole value under our name and replace it when it differs; nothing else in the
// document is read or written.
//
// The one field carried over is `enabled`. agy's `/hooks` command toggles it, and
// switching a hook back on that a person switched off would override an explicit
// choice. Identical bytes report changed=false, so a re-run writes nothing.
func MergeNamedHooks(doc map[string]any, name string, cmds []HookCommand) (map[string]any, bool, error) {
	if name == "" {
		return nil, false, fmt.Errorf("a named hook needs a name: it is the only thing that marks it ours")
	}
	if doc == nil {
		doc = map[string]any{}
	}

	// Sorted by event for byte-stable output; stable, so two hooks on one event
	// keep their declared order.
	sorted := append([]HookCommand(nil), cmds...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Event < sorted[j].Event })

	want := map[string]any{}
	for _, c := range sorted {
		if c.Event == "" || c.Command == "" {
			return nil, false, fmt.Errorf("hook for event %q has no command", c.Event)
		}
		handler := map[string]any{"type": "command", "command": c.Command}
		if c.Timeout > 0 {
			handler["timeout"] = c.Timeout
		}
		entries, _ := want[c.Event].([]any)
		if namedGroupedEvents[c.Event] {
			matcher := c.Matcher
			if matcher == "" {
				matcher = "*"
			}
			entries = append(entries, map[string]any{"matcher": matcher, "hooks": []any{handler}})
		} else {
			entries = append(entries, handler)
		}
		want[c.Event] = entries
	}

	if current, ok := doc[name].(map[string]any); ok {
		if enabled, has := current["enabled"]; has {
			want["enabled"] = enabled
		}
	}

	if got, ok := doc[name].(map[string]any); ok && sameHook(got, want) {
		return doc, false, nil
	}
	doc[name] = want
	return doc, true, nil
}

// RetireHooks removes OUR entry for (event, id) from a claude-shaped settings
// document, reporting whether anything changed.
//
// It is for a hook that moved to another file. Leaving the old entry would keep it
// firing wherever the old file is still read, with a payload shape it no longer
// parses - the agy gate sat in Gemini CLI's settings.json for exactly that
// reason. Only what isOurs recognises is removed; a group that ours emptied goes
// with it, and an event with no groups left goes too, while a foreign hook in the
// same group, or a foreign group on the same event, is never touched.
func RetireHooks(doc map[string]any, event, id string) (map[string]any, bool) {
	hooks, _ := doc["hooks"].(map[string]any)
	groups, _ := hooks[event].([]any)
	if len(groups) == 0 {
		return doc, false
	}

	changed := false
	kept := make([]any, 0, len(groups))
	for _, g := range groups {
		group, ok := g.(map[string]any)
		if !ok {
			kept = append(kept, g)
			continue
		}
		inner, _ := group["hooks"].([]any)
		left := make([]any, 0, len(inner))
		for _, h := range inner {
			if obj, ok := h.(map[string]any); ok && isOurs(obj, id, "") {
				changed = true
				continue
			}
			left = append(left, h)
		}
		switch {
		case len(left) == len(inner):
			kept = append(kept, g) // nothing of ours in it: leave the group exactly as it was
		case len(left) == 0:
			// the group held only our entry: drop it whole
		default:
			group["hooks"] = left
			kept = append(kept, group)
		}
	}
	if !changed {
		return doc, false
	}
	if len(kept) == 0 {
		delete(hooks, event)
	} else {
		hooks[event] = kept
	}
	return doc, true
}
