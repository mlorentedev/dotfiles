package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func gatePersona(skills ...SkillBinding) *Persona {
	return &Persona{Name: "reviewer", Skills: skills}
}

// AC3's core, with no harness installed — the agnosticism requirement made
// executable: the decision is a pure function over normalised input.
func TestGateBlocksOnAnUnconsumedBlockingSkill(t *testing.T) {
	r := Decide(GateInput{
		Persona: gatePersona(
			SkillBinding{ID: "adversarial-review", Enforce: EnforceBlock},
			SkillBinding{ID: "audit", Enforce: EnforceBlock},
		),
		Call:     ToolCall{Tool: "Bash"},
		Consumed: map[string]bool{"audit": true},
	})
	if r.Decision != Block {
		t.Fatalf("want Block, got %v (%s)", r.Decision, r.Reason)
	}
	if len(r.Missing) != 1 || r.Missing[0] != "adversarial-review" {
		t.Errorf("Missing = %v, want [adversarial-review]", r.Missing)
	}
	if !strings.Contains(r.Reason, "adversarial-review") {
		t.Errorf("the reason must name what to do: %q", r.Reason)
	}
}

// AC4 — warn emits and does NOT block, asserted on the same path so the two
// cannot quietly collapse into one behaviour.
func TestGateWarnDoesNotBlock(t *testing.T) {
	r := Decide(GateInput{
		Persona:  gatePersona(SkillBinding{ID: "cyclomatic-complexity", Enforce: EnforceWarn}),
		Call:     ToolCall{Tool: "Bash"},
		Consumed: map[string]bool{},
	})
	if r.Decision != Allow {
		t.Fatalf("a warn skill must never block, got %v", r.Decision)
	}
	if len(r.Warned) != 1 || r.Warned[0] != "cyclomatic-complexity" {
		t.Errorf("Warned = %v, want [cyclomatic-complexity]", r.Warned)
	}
}

// The migration guarantee: a skill with no declared severity is neither
// enforced nor warned on every tool call. Picking either would be the default
// this design refuses to pick.
func TestGateIgnoresUnmigratedSkills(t *testing.T) {
	r := Decide(GateInput{
		Persona:  gatePersona(SkillBinding{ID: "test", Enforce: EnforceUnset}),
		Call:     ToolCall{Tool: "Bash"},
		Consumed: map[string]bool{},
	})
	if r.Decision != Allow {
		t.Errorf("an unmigrated skill must not block, got %v", r.Decision)
	}
	if len(r.Warned) != 0 || len(r.Missing) != 0 {
		t.Errorf("an unmigrated skill must not be reported here: missing=%v warned=%v", r.Missing, r.Warned)
	}
}

// Blocking the skill invocation itself would deadlock the session: the gate
// would forbid the only action that could satisfy it.
func TestGateNeverBlocksTheSkillInvocationItself(t *testing.T) {
	r := Decide(GateInput{
		Persona:  gatePersona(SkillBinding{ID: "audit", Enforce: EnforceBlock}),
		Call:     ToolCall{Tool: "Skill", Skill: "audit"},
		Consumed: map[string]bool{},
	})
	if r.Decision != Allow {
		t.Fatalf("invoking a skill must always be allowed, or the gate deadlocks: %v", r.Decision)
	}
}

// The deadlock the PR reviewer on #1272 found: a WELL-FORMED payload naming the
// skill primitive but carrying no readable argument.
//
// The original guard keyed on the skill's NAME, so an empty name fell through to
// the enforcement path and blocked the invocation — permanently, because a
// blocked call can never record consumption. The guard has to key on the tool's
// IDENTITY. The earlier deadlock test passed only because it always supplied a
// name, which is the case that was never in danger.
func TestGateNeverBlocksASkillToolWithAnUnreadableName(t *testing.T) {
	r := Decide(GateInput{
		Persona:  gatePersona(SkillBinding{ID: "audit", Enforce: EnforceBlock}),
		Call:     ToolCall{Tool: "Skill", Skill: "", IsSkillTool: true},
		Consumed: map[string]bool{},
	})
	if r.Decision != Allow {
		t.Fatalf("a skill invocation with an unreadable name must still be allowed, got %v — this deadlocks the session", r.Decision)
	}
}

// Ambiguity resolves to Allow. A gate that blocks when it does not understand
// its input becomes a gate whose normal state is red.
func TestGateAllowsWhenItHasNoPersona(t *testing.T) {
	if r := Decide(GateInput{Call: ToolCall{Tool: "Bash"}}); r.Decision != Allow {
		t.Fatalf("no persona must resolve to Allow, got %v", r.Decision)
	}
}

// Once the blocking skills are consumed the gate opens — otherwise it is not a
// gate, it is a wall.
func TestGateOpensOnceConsumed(t *testing.T) {
	r := Decide(GateInput{
		Persona:  gatePersona(SkillBinding{ID: "audit", Enforce: EnforceBlock}),
		Call:     ToolCall{Tool: "Bash"},
		Consumed: map[string]bool{"audit": true},
	})
	if r.Decision != Allow {
		t.Fatalf("want Allow after consumption, got %v (%s)", r.Decision, r.Reason)
	}
}

func TestGateStateRoundTrips(t *testing.T) {
	dir := t.TempDir()
	p := StatePath(dir, "sess-123")

	if got := LoadConsumed(p); len(got) != 0 {
		t.Errorf("a missing state file must read as an empty set, got %v", got)
	}
	if err := RecordConsumed(p, "audit"); err != nil {
		t.Fatal(err)
	}
	if err := RecordConsumed(p, "audit"); err != nil {
		t.Fatalf("recording twice must be idempotent: %v", err)
	}
	if got := LoadConsumed(p); !got["audit"] || len(got) != 1 {
		t.Errorf("LoadConsumed = %v, want {audit:true}", got)
	}
}

// Losing or corrupting state must never turn into a blocked session: the record
// is a cache of what happened, not a source of authority.
func TestGateCorruptStateReadsAsEmptyNotAsAnError(t *testing.T) {
	dir := t.TempDir()
	p := StatePath(dir, "sess")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadConsumed(p); len(got) != 0 {
		t.Errorf("corrupt state must read empty, got %v", got)
	}
}

// The collision the PR reviewer on #1272 found: character-mapping alone flattens
// `a/b` and `a.b` to the same name, so one session's consumption record would
// open another session's gate. UUIDs never collide, but "it does not happen with
// well-behaved input" is not a property a path builder should rest on.
func TestGateStatePathDoesNotCollideAcrossDistinctSessions(t *testing.T) {
	dir := t.TempDir()
	seen := map[string]string{}
	for _, sid := range []string{"a/b", "a.b", "a_b", "a:b", "a b", "", "..", "x", "X"} {
		p := StatePath(dir, sid)
		if prev, dup := seen[p]; dup {
			t.Errorf("session ids %q and %q map to the same state file %s", prev, sid, p)
		}
		seen[p] = sid
	}
}

// A session id is attacker-adjacent input on some harnesses, and it lands in a
// filesystem path.
func TestGateStatePathContainsNoTraversal(t *testing.T) {
	dir := t.TempDir()
	for _, sid := range []string{"../../etc/passwd", "a/b", "..", "x\x00y"} {
		got := StatePath(dir, sid)
		rel, err := filepath.Rel(dir, got)
		if err != nil || filepath.IsAbs(rel) || strings.Contains(rel, "..") {
			t.Errorf("session id %q escaped the state dir: %s", sid, got)
		}
	}
}

// TestAnEmptyScopeHasNoStateFile pins the fix for the shared ledger.
//
// scopeKey("") is "unknown-e3b0c442", the digest of the empty string, so every
// payload that named no session used to share ONE consumption ledger and ONE
// dispatch map. The first skill any of them invoked would then satisfy the gate
// for all the rest, and a dispatch recorded by one would answer another's
// lookup. Nothing durable that gates a call may be keyed by nothing, so the empty
// scope has no path, and writing through it is a no-op rather than an error or a
// file.
func TestAnEmptyScopeHasNoStateFile(t *testing.T) {
	dir := t.TempDir()

	if got := StatePath(dir, ""); got != "" {
		t.Errorf("StatePath for an empty scope = %q, want no path", got)
	}
	if got := DispatchPath(dir, ""); got != "" {
		t.Errorf("DispatchPath for an empty session = %q, want no path", got)
	}

	if err := RecordConsumed(StatePath(dir, ""), "audit"); err != nil {
		t.Errorf("recording through the empty path must be a quiet no-op, got %v", err)
	}
	if err := RecordDispatch(DispatchPath(dir, ""), "n", "reviewer"); err != nil {
		t.Errorf("recording a dispatch through the empty path must be a quiet no-op, got %v", err)
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Errorf("nothing may be written for an empty scope, found %v (err %v)", entries, err)
	}

	if got := LoadConsumed(StatePath(dir, "")); len(got) != 0 {
		t.Errorf("nothing can be read back through the empty path, got %v", got)
	}
	if got := LoadDispatched(DispatchPath(dir, "")); len(got) != 0 {
		t.Errorf("no dispatch can be read back through the empty path, got %v", got)
	}
}

// The agent id is unique per invocation, not per session, so it cannot stand in
// for a session that is missing: a ledger keyed by it alone would be one file per
// dispatch that nothing could ever find again.
func TestConsumptionScopeNeedsASession(t *testing.T) {
	for _, tc := range []struct {
		name string
		call ToolCall
		want string
	}{
		{"no session and no agent", ToolCall{}, ""},
		{"an agent but no session", ToolCall{AgentID: "a1"}, ""},
		{"a session alone", ToolCall{SessionID: "s1"}, "s1"},
		{"a session and an agent", ToolCall{SessionID: "s1", AgentID: "a1"}, "s1-a1"},
		// `_unscoped` is where calls with NO session are journaled, so a payload that
		// claims it as its session id names no session at all.
		{"the id reserved for the sessionless journal", ToolCall{SessionID: UnscopedScope}, ""},
		{"the reserved id with an agent", ToolCall{SessionID: UnscopedScope, AgentID: "a1"}, ""},
		// `_unparsed` is the journal for payloads the gate could not read, so it is
		// reserved the same way (HARNESS-149).
		{"the id reserved for unreadable payloads", ToolCall{SessionID: UnparsedScope}, ""},
		{"the unparsed id with an agent", ToolCall{SessionID: UnparsedScope, AgentID: "a1"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.call.ConsumptionScope(); got != tc.want {
				t.Errorf("ConsumptionScope() = %q, want %q", got, tc.want)
			}
		})
	}
}

// #1510: the three ALLOW states must be distinguishable from each other.
//
// They shared one sentence — "all blocking skills consumed" — which was
// vacuously true (no persona carries `enforce: block`, so the predicate held in
// all 11526 decisions ever recorded) and therefore discriminated nothing. An
// inert gate and a passing gate wrote the same line, which is precisely what
// architect's and shipper's shipped records cite as the reason a merged
// migration cannot be trusted as an enforced one.
//
// Asserted as PAIRWISE DISTINCTNESS rather than against three literal strings:
// the property that matters is that a reader can tell the states apart, and
// pinning the wording would make this a chore on every rephrasing without
// testing anything more.
func TestGateAllowReasonsAreDistinguishable(t *testing.T) {
	call := ToolCall{Tool: "Bash"}

	nothingToEnforce := Decide(GateInput{
		Persona:  gatePersona(SkillBinding{ID: "helm"}, SkillBinding{ID: "terraform"}),
		Call:     call,
		Consumed: map[string]bool{},
	})
	skippedEverything := Decide(GateInput{
		Persona:  gatePersona(SkillBinding{ID: "audit", Enforce: EnforceWarn}),
		Call:     call,
		Consumed: map[string]bool{},
	})
	satisfied := Decide(GateInput{
		Persona:  gatePersona(SkillBinding{ID: "audit", Enforce: EnforceWarn}),
		Call:     call,
		Consumed: map[string]bool{"audit": true},
	})

	for _, r := range []GateResult{nothingToEnforce, skippedEverything, satisfied} {
		if r.Decision != Allow {
			t.Fatalf("all three states allow; got %v (%s)", r.Decision, r.Reason)
		}
	}

	seen := map[string][]string{}
	for name, r := range map[string]GateResult{
		"nothing to enforce": nothingToEnforce,
		"skipped everything": skippedEverything,
		"satisfied":          satisfied,
	} {
		seen[r.Reason] = append(seen[r.Reason], name)
	}
	for reason, names := range seen {
		if len(names) > 1 {
			t.Errorf("these states are indistinguishable, all reporting %q: %v — "+
				"an inert gate reads exactly like a passing one", reason, names)
		}
	}

	// The state that hid best, called out on its own: a persona declaring no
	// severity enforces nothing, and the line a human reads must say so.
	if !strings.Contains(nothingToEnforce.Reason, "nothing to enforce") {
		t.Errorf("a persona with no severities must say enforcement is off, got %q", nothingToEnforce.Reason)
	}
	// And the warn state must name what was skipped, since Warned is otherwise
	// the only record of it.
	if !strings.Contains(skippedEverything.Reason, "audit") {
		t.Errorf("the warn reason must name the skipped skill, got %q", skippedEverything.Reason)
	}
}
