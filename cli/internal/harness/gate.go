package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Decision is what the gate tells the harness to do.
type Decision int

const (
	// Allow lets the tool call proceed (exit 0).
	Allow Decision = iota
	// Block stops it (exit 2 on claude, a thrown error on pi, deny on opencode).
	Block
)

// GateResult is a decision plus the reason, which is the part a human reads.
type GateResult struct {
	Decision Decision
	Reason   string
	// Missing are the blocking skills not yet consumed this session.
	Missing []string
	// Warned are `enforce: warn` skills not yet consumed — reported, not enforced.
	Warned []string
}

// ToolCall is the harness-neutral slice of a tool-call event.
//
// Each harness's payload is normalised into this before any decision is made, so
// the decision logic is written once and tested with no harness installed —
// which is the agnosticism requirement, not a convenience.
type ToolCall struct {
	SessionID string
	// Tool is the harness's name for the tool about to run.
	Tool string
	// Skill is the skill being invoked, when Tool is the harness's skill
	// primitive. Empty otherwise — INCLUDING when the tool IS the skill
	// primitive but its argument could not be read, which is why IsSkillTool
	// exists separately.
	Skill string
	// AgentType names the dispatched persona this call belongs to, when the
	// harness says so. Empty on a main-thread call, which means "no persona in
	// scope" and allows — never an error.
	AgentType string
	// AgentID scopes the consumption ledger to ONE dispatched persona.
	//
	// It matters because the harness reuses the parent's session id inside a
	// subagent: keying consumption by session alone would let the parent's skill
	// runs satisfy the reviewer's gate, and every persona dispatched in one
	// session would share a ledger. Consumption is a claim about who has done
	// what, so it is scoped to whoever the harness says is acting. Empty falls
	// back to the session, which is right for a main-thread call.
	AgentID string
	// DispatchName and DispatchType are read from the Agent tool's own
	// arguments when THIS call is a dispatch, and are how the gate learns what a
	// named subagent really is (HARNESS-109). They are set on the PARENT's call,
	// never on the child's, and are the only tool-input values other than Skill
	// that the gate reads at all — both schema-bounded identifiers, validated by
	// ValidDispatchName before anything is written. See dispatch.go.
	DispatchName string
	DispatchType string
	// IsSkillTool reports that the tool is the harness's skill primitive,
	// whether or not the skill's NAME was readable.
	//
	// Without it the gate deadlocks on a WELL-FORMED payload with a missing
	// argument: `Skill` comes back empty, the call falls through to the
	// enforcement path, and the gate blocks the very invocation that would
	// satisfy it — permanently, because a blocked call can never record
	// consumption. Raised by the PR reviewer on #1272 and reproduced. The
	// deadlock guard was checking the skill's NAME when it had to check the
	// tool's IDENTITY.
	IsSkillTool bool
}

// GateInput is everything a decision needs.
type GateInput struct {
	Persona  *Persona
	Call     ToolCall
	Consumed map[string]bool
}

// Decide is the whole policy, as a pure function.
//
// IT NEVER ERRORS, and that is a cross-harness requirement rather than style.
// The harnesses disagree about what a hook failure means: claude's hooks are
// contractually forbidden from crashing a session (fail-open), while pi
// documents that "`tool_call` errors block the tool (fail-safe)" — fail-closed.
// The same malformed payload would therefore allow on one harness and block on
// the other. Returning an explicit decision every time keeps that asymmetry out
// of the result: the harness never sees an error, so its error semantics never
// apply.
//
// Ambiguity resolves to Allow. A gate that blocks when it does not understand
// its input becomes a gate whose normal state is red — the failure already
// recorded in this repository about a reviewer whose decline stopped meaning
// anything.
func Decide(in GateInput) GateResult {
	if in.Persona == nil {
		return GateResult{Decision: Allow, Reason: "no persona in scope"}
	}

	// Invoking a skill is never gated: that is the act the gate exists to
	// require, and blocking it would deadlock the session. Keyed on the TOOL,
	// not on whether the skill's name parsed — an unreadable argument must never
	// turn the one unblockable action into a blocked one.
	if in.Call.IsSkillTool || in.Call.Skill != "" {
		return GateResult{Decision: Allow, Reason: "skill invocation"}
	}

	var missing, warned []string
	for _, s := range in.Persona.Skills {
		if in.Consumed[s.ID] {
			continue
		}
		switch s.Enforce {
		case EnforceBlock:
			missing = append(missing, s.ID)
		case EnforceWarn:
			warned = append(warned, s.ID)
		case EnforceUnset:
			// A skill carrying no declared severity is NOT enforced and NOT
			// warned about here. It is surfaced by `dotf doctor` instead, where
			// it reads as a migration gap rather than as noise on every tool
			// call. Silently treating it as either would be the default this
			// design refuses to pick.
		}
	}
	sort.Strings(missing)
	sort.Strings(warned)

	if len(missing) > 0 {
		return GateResult{
			Decision: Block,
			Missing:  missing,
			Warned:   warned,
			Reason: fmt.Sprintf("persona %q requires %s before other tools; invoke %s first",
				in.Persona.Name, plural(len(missing), "skill"), strings.Join(missing, ", ")),
		}
	}
	return GateResult{Decision: Allow, Warned: warned, Reason: "all blocking skills consumed"}
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// --- session state -----------------------------------------------------------

// consumedState is what a session has invoked so far.
type consumedState struct {
	Skills map[string]bool `json:"skills"`
}

// StatePath is where a session's consumption record lives.
//
// Under the state dir rather than beside the persona records: this is per-run,
// per-machine data with no business in a git-tracked tree, and a stale file
// simply means an over-permissive gate for one session rather than corrupted
// configuration.
// A DIGEST IS APPENDED, not merely a sanitised name. Character-mapping alone
// collides: `a/b` and `a.b` both flatten to `a_b`, so one session's consumption
// record would open another session's gate. Raised by the PR reviewer on #1272.
// Well-behaved harnesses send UUIDs and would never hit it, but a session id is
// attacker-adjacent input that lands in a filesystem path, and "it does not
// happen with well-behaved input" is not a property a path builder should rely
// on. The readable prefix is kept so the directory stays diagnosable by eye.
func StatePath(stateDir, sessionID string) string {
	return filepath.Join(stateDir, "gate", scopeKey(sessionID)+".json")
}

// scopeKey turns a scope into the readable-prefix-plus-digest filename stem that
// both the consumption ledger and the decision journal are keyed by.
//
// It is shared rather than duplicated because the two must agree: a scope whose
// ledger and journal disagreed would report decisions for one dispatch against
// another's consumption, and the digest is the part that makes either of them
// safe. Extracted when the journal landed; the behaviour is unchanged.
func scopeKey(scope string) string {
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, scope)
	if len(safe) > 48 {
		safe = safe[:48]
	}
	if safe == "" {
		safe = "unknown"
	}
	sum := sha256.Sum256([]byte(scope))
	return safe + "-" + hex.EncodeToString(sum[:4])
}

// LoadConsumed reads a session's consumed skills. A missing or unreadable file
// is an empty set, never an error: state is a cache of what happened, and losing
// it must not turn into a blocked session.
func LoadConsumed(path string) map[string]bool {
	out := map[string]bool{}
	raw, err := os.ReadFile(path) // #nosec G304 -- path is built by StatePath from the state dir
	if err != nil {
		return out
	}
	var st consumedState
	if json.Unmarshal(raw, &st) != nil {
		return out
	}
	for k, v := range st.Skills {
		if v {
			out[k] = true
		}
	}
	return out
}

// RecordConsumed adds a skill to the session's set.
func RecordConsumed(path, skill string) error {
	if skill == "" {
		return nil
	}
	current := LoadConsumed(path)
	if current[skill] {
		return nil
	}
	current[skill] = true
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(consumedState{Skills: current})
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

// ConsumptionScope is the key a call's consumption ledger is stored under.
//
// The dispatched persona when the harness names one, the session otherwise. Both
// are needed: without the agent scope, every persona dispatched in one session
// shares a ledger and the first one's skill runs silently satisfy the rest;
// without the session fallback, a main-thread call — which carries no agent —
// would have no ledger at all.
//
// SEPARATION IS ONLY AS GOOD AS AgentID'S UNIQUENESS, which is documented as a
// per-invocation id and NOT YET MEASURED here — the same envelope as the field
// names themselves, raised as a distinct point by the reviewer on #1410. If the
// harness were instead to send a value stable per persona (a name, a hash of the
// type), two dispatches of `reviewer` in one session would share this key and
// the second would inherit the first's consumption.
//
// That failure is over-permissive, never over-strict: a shared ledger can only
// skip a gate, never raise one, so it costs enforcement and cannot block a
// session. It is checked by the same measurement that confirms the field names —
// dispatch one persona TWICE in a session and confirm the second is gated again
// — and that check is a precondition of promoting any skill to `enforce: block`,
// not of this function being correct for the single-dispatch case.
func (c ToolCall) ConsumptionScope() string {
	if c.AgentID != "" {
		return c.SessionID + "-" + c.AgentID
	}
	return c.SessionID
}
