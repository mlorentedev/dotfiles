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

// A session id is attacker-adjacent input on some harnesses, and it lands in a
// filesystem path.
func TestGateStatePathContainsNoTraversal(t *testing.T) {
	dir := t.TempDir()
	for _, sid := range []string{"../../etc/passwd", "a/b", "", "..", "x\x00y"} {
		got := StatePath(dir, sid)
		rel, err := filepath.Rel(dir, got)
		if err != nil || filepath.IsAbs(rel) || strings.Contains(rel, "..") {
			t.Errorf("session id %q escaped the state dir: %s", sid, got)
		}
	}
}
