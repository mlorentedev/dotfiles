package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The dispatch map implements HARNESS-109 (#1434): it is how the gate learns
// which PERSONA a named subagent actually is.
//
// THE PROBLEM IT SOLVES IS THAT THE PAYLOAD CANNOT ANSWER IT. `agent_type`
// carries the caller-supplied NAME when a dispatch is named, so the persona
// lookup misses and the gate fails open — measured on 271 of the 274 journal
// records that carried an agent_type. Issue #1434 proposed falling back to a
// second payload field carrying the true type; that direction is refuted, not
// deferred. Read out of the Claude Code 2.1.260 executable's own payload
// builder, the complete base field set is session_id, transcript_path, cwd,
// scratchpad_dir, prompt_id, permission_mode, agent_id, agent_type and effort.
// `agent_type` is the only type-carrying field, and `agent_id` is `a<name>-
// <hash>` — the name again. There is nothing to fall back to.
//
// WHAT IS AVAILABLE IS THE DISPATCH ITSELF. The gate runs on the PARENT's call
// too, and that call is the Agent tool, whose arguments carry both the name and
// the true type. PreToolUse is synchronous — the tool does not execute until the
// hook exits — so the parent's hook has always completed before the child
// exists. The ordering is structural, not a race won by 3.5 seconds.
//
// IT LIVES IN THE STATE DIR, NEVER IN THE JOURNAL. `DecisionRecord` carries no
// tool input, and that stays true: this file is a separate per-session cache
// beside the consumption ledger. What it stores are two schema-bounded
// identifiers, validated against the dispatch-name pattern before being written,
// so no free text — and therefore no file content, shell command or credential —
// can reach disk through this path. The precedent is `skillArg`, which has read
// `tool_input.skill` and recorded it as a NAME since #1435.

// dispatchNamePattern is the Agent tool's own constraint on a dispatch name.
//
// Validating against it is what keeps this path from being a free-text sink: a
// value that does not match is not written at all, and the call still allows.
// The same pattern is applied to the type, which is drawn from the same
// argument object and is equally caller-supplied.
var dispatchNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// maxDispatchEntries bounds one session's map.
//
// A hook that appends without a limit is a way to fill a disk, which would then
// fail the very writes the gate's fail-open discipline assumes are harmless —
// the reasoning behind maxDecisionBytes, applied to the other file the gate
// writes. Past the cap nothing more is recorded, so an unknown name resolves to
// nothing and the gate behaves exactly as it did before this existed.
const maxDispatchEntries = 512

// dispatchState is the on-disk shape: dispatched identity -> true agent type.
type dispatchState struct {
	Types map[string]string `json:"types"`
}

// ValidDispatchName reports whether a value may be written to the map.
func ValidDispatchName(s string) bool {
	return dispatchNamePattern.MatchString(s)
}

// DispatchPath is where one session's map lives.
//
// KEYED BY SESSION, NOT BY THE CONSUMPTION SCOPE, and the difference is what
// makes the lookup work: a subagent reuses its parent's session id, so the
// parent writes and the child reads the same file. Scoping it the way the
// ledger is scoped would file the parent's write under the parent's agent id,
// where the child would never look — and it would break a grandchild besides,
// since every generation shares the one session id and no other key.
//
// It reuses scopeKey, and therefore the same collision digest as the ledger and
// the journal, for the reason documented there: character-mapping alone
// collides, and a collision here would answer one session's lookup with another
// session's dispatch — a wrong persona, which is worse than none.
func DispatchPath(stateDir, sessionID string) string {
	return filepath.Join(stateDir, "gate", scopeKey(sessionID)+".dispatch.json")
}

// LoadDispatched reads a session's map. A missing or unreadable file is an empty
// map, never an error: like the consumption ledger, this is a cache of what
// happened, and losing it must degrade to the pre-existing allow.
func LoadDispatched(path string) map[string]string {
	out := map[string]string{}
	raw, err := os.ReadFile(path) // #nosec G304 -- path is built by DispatchPath from the state dir
	if err != nil {
		return out
	}
	var st dispatchState
	if json.Unmarshal(raw, &st) != nil {
		return out
	}
	for k, v := range st.Types {
		if k != "" && v != "" {
			out[k] = v
		}
	}
	return out
}

// RecordDispatch remembers that `name` was dispatched as `agentType`.
//
// AN UNNAMED DISPATCH IS RECORDED TOO, under its own type. That looks redundant
// and is the reason no hardcoded list of built-in agents is needed anywhere:
// presence in this map means "the gate WITNESSED this dispatch and knows what it
// really is". A witnessed dispatch whose true type is not a persona —
// `general-purpose`, `Explore`, `Plan` — is a correct no-role, quietly, exactly
// like a main-thread call. An agent_type absent from the map is the genuinely
// unknown case and stays loud. Without this line the two would be
// indistinguishable again, one level further in.
//
// Latest-wins on a reused name: the most recent dispatch is the one whose calls
// are arriving.
func RecordDispatch(path, name, agentType string) error {
	name = strings.TrimSpace(name)
	agentType = strings.TrimSpace(agentType)
	if agentType == "" {
		// Nothing to remember: without a type the entry could not answer the
		// only question this map is asked.
		return nil
	}
	if name == "" {
		name = agentType
	}
	if !ValidDispatchName(name) || !ValidDispatchName(agentType) {
		return nil
	}

	current := LoadDispatched(path)
	if current[name] == agentType {
		return nil
	}
	if len(current) >= maxDispatchEntries && current[name] == "" {
		return nil
	}
	current[name] = agentType

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	raw, err := json.Marshal(dispatchState{Types: current})
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}
