package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// The regression this file exists for, measured 2026-08-26.
//
// `normaliseToolCall` used to return a zero ToolCall on a payload it could not
// parse, and the doc comment claimed Decide would resolve that to Allow. Decide
// does no such thing: given a valid persona and nothing consumed it BLOCKS, so a
// malformed payload blocked every tool call. The unit tests over Decide all
// passed, because the defect lived in the seam between them — prose asserting a
// behaviour the code did not have, visible only by running the binary.
//
// A gate that blocks on input it cannot read blocks on every harness upgrade,
// which is the "normal state is red" failure this design refuses.
func TestNormaliseReportsWhetherThePayloadWasUnderstood(t *testing.T) {
	for _, tc := range []struct {
		name, harnessName, payload string
		wantOK                     bool
		wantTool, wantSkill, wantS string
	}{
		{name: "claude tool call", harnessName: "claude",
			payload:  `{"session_id":"s1","tool_name":"Bash","tool_input":{"command":"ls"}}`,
			wantOK:   true,
			wantTool: "Bash", wantS: "s1"},
		{name: "claude skill invocation", harnessName: "claude",
			payload:   `{"session_id":"s1","tool_name":"Skill","tool_input":{"skill":"audit"}}`,
			wantOK:    true,
			wantTool:  "Skill",
			wantSkill: "audit", wantS: "s1"},
		// pi and opencode run in-process TypeScript, not command hooks, so the
		// wrapper this repository generates for them emits the canonical shape.
		{name: "pi via the generated wrapper", harnessName: "pi",
			payload:  `{"session":"s2","tool":"bash"}`,
			wantOK:   true,
			wantTool: "bash", wantS: "s2"},
		{name: "opencode via the generated wrapper", harnessName: "opencode",
			payload:  `{"session":"s3","tool":"bash"}`,
			wantOK:   true,
			wantTool: "bash", wantS: "s3"},
		{name: "canonical skill invocation", harnessName: "pi",
			payload:   `{"session":"s2","tool":"skill","skill":"audit"}`,
			wantOK:    true,
			wantTool:  "skill",
			wantSkill: "audit", wantS: "s2"},
		// agy uses claude's command-hook shape: ~/.gemini/settings.json declares
		// BeforeTool in that exact format, so it takes the default branch.
		{name: "agy command hook", harnessName: "agy",
			payload:  `{"session_id":"s4","tool_name":"Bash","tool_input":{}}`,
			wantOK:   true,
			wantTool: "Bash", wantS: "s4"},

		// Every one of these must report NOT understood, so the caller allows.
		{name: "not json", harnessName: "claude", payload: `not json at all`},
		{name: "empty", harnessName: "claude", payload: ``},
		{name: "json without a tool", harnessName: "claude", payload: `{"session_id":"s1"}`},
		{name: "command shape sent to a wrapper harness", harnessName: "pi", payload: `{"session_id":"s1","tool_name":"Bash"}`},
		{name: "canonical shape sent to a command harness", harnessName: "claude", payload: `{"session":"s1","tool":"bash"}`},
		{name: "json array", harnessName: "claude", payload: `[1,2,3]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			call, ok := normaliseToolCall(tc.harnessName, []byte(tc.payload))
			if ok != tc.wantOK {
				t.Fatalf("understood = %v, want %v (payload %q)", ok, tc.wantOK, tc.payload)
			}
			if !tc.wantOK {
				return
			}
			if call.Tool != tc.wantTool || call.Skill != tc.wantSkill || call.SessionID != tc.wantS {
				t.Errorf("got {tool:%q skill:%q session:%q}, want {tool:%q skill:%q session:%q}",
					call.Tool, call.Skill, call.SessionID, tc.wantTool, tc.wantSkill, tc.wantS)
			}
		})
	}
}

// And the seam itself: an unparseable payload must not reach Decide as "a real
// call with nothing consumed". Pinned end to end so the two halves cannot drift
// back apart.
func TestUnreadablePayloadNeverReachesDecideAsARealCall(t *testing.T) {
	persona := &harness.Persona{
		Name:   "reviewer",
		Skills: []harness.SkillBinding{{ID: "adversarial-review", Enforce: harness.EnforceBlock}},
	}
	call, ok := normaliseToolCall("claude", []byte("not json"))
	if ok {
		t.Fatal("garbage must not be reported as understood")
	}
	// The proof that the `ok` flag is load-bearing: were the caller to ignore it
	// and hand this straight to Decide, the result would be Block.
	if r := harness.Decide(harness.GateInput{Persona: persona, Call: call}); r.Decision != harness.Block {
		t.Fatal("Decide no longer blocks on an empty call — this test's premise is stale, re-check the caller")
	}
}

// The skill name is read from whichever key a harness uses, and only when the
// tool IS the skill primitive. A tool merely named in the arguments is not an
// invocation.
func TestSkillArgOnlyReadsTheSkillPrimitive(t *testing.T) {
	if got := skillArg("Bash", map[string]any{"skill": "audit"}); got != "" {
		t.Errorf("a non-skill tool must yield no skill, got %q", got)
	}
	for _, key := range []string{"skill", "name", "command"} {
		if got := skillArg("skill", map[string]any{key: "audit"}); got != "audit" {
			t.Errorf("key %q: got %q, want audit", key, got)
		}
	}
	if got := skillArg("Skill", map[string]any{"skill": "   "}); got != "" {
		t.Errorf("a blank skill name must yield no skill, got %q", got)
	}
}

// TestLoadGatePersonaSaysSoWhenARoleDoesNotResolve pins the fix for a guard that
// failed OPEN and reported health.
//
// Measured on the shipped binary before the fix:
//
//	$ printf '{"tool_name":"Bash","session_id":"t1"}' | dotf harness gate \
//	      --harness claude --role reviewr --repo-root .
//	$ echo $?
//	0
//
// No output at all. A typo in the role, a renamed record, or a --repo-root that
// resolved somewhere without harness/agents disabled enforcement for the whole
// session, and nothing anywhere said so. Allowing is the correct DECISION —
// blocking every call on a config error is the worse failure — but silence is
// not, because it is indistinguishable from a session with enforcement working.
//
// Table-driven per AGENTS.md, one row per branch of the resolution.
func TestLoadGatePersonaSaysSoWhenARoleDoesNotResolve(t *testing.T) {
	root := repoRootForTest(t)

	for _, tc := range []struct {
		name     string
		role     string
		repoRoot string
		wantNil  bool
		wantSaid bool
	}{
		{
			name: "no role asked for is silent — nothing failed to resolve",
			role: "", repoRoot: root, wantNil: true, wantSaid: false,
		},
		{
			name: "a role that does not exist is allowed AND announced",
			role: "reviewr", repoRoot: root, wantNil: true, wantSaid: true,
		},
		{
			name: "a repo root without harness/agents is allowed AND announced",
			role: "reviewer", repoRoot: t.TempDir(), wantNil: true, wantSaid: true,
		},
		{
			name: "a role that resolves is silent",
			role: "reviewer", repoRoot: root, wantNil: false, wantSaid: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			got := loadGatePersona(&buf, tc.repoRoot, tc.role)

			if (got == nil) != tc.wantNil {
				t.Errorf("persona nil = %v, want %v", got == nil, tc.wantNil)
			}
			// Always allow-shaped: this function never causes a block.
			said := buf.Len() > 0
			if said != tc.wantSaid {
				t.Errorf("announced = %v, want %v (stderr=%q)", said, tc.wantSaid, buf.String())
			}
			if tc.wantSaid && !strings.Contains(buf.String(), "ENFORCEMENT IS OFF") {
				t.Errorf("the message does not say enforcement is off: %q", buf.String())
			}
		})
	}
}

// TestGateResolvesTheRoleFromThePayload is the fix for the reason the whole
// binding chain decided nothing.
//
// MEASURED 2026-08-31, before this: the manifest emits `harness gate --harness
// claude` with NO --role. loadGatePersona returned nil on an empty role, Decide
// answered "no persona in scope", and every tool call was allowed regardless of
// what any skill declared. bind, the hook, the gate and Decide were all live and
// enforced nothing.
//
// Claude documents `agent_type` on every hook event fired inside a subagent, and
// nothing on a main-thread call. So the harness already knows what the gate was
// missing, and no session-state mechanism was needed to carry it.
func TestGateResolvesTheRoleFromThePayload(t *testing.T) {
	for _, tc := range []struct {
		name     string
		flag     string
		payload  string
		wantRole string
	}{
		{
			name:     "the payload's agent_type becomes the role",
			payload:  `{"tool_name":"Bash","session_id":"s","agent_type":"reviewer"}`,
			wantRole: "reviewer",
		},
		{
			name:     "a main-thread call names no persona, which allows",
			payload:  `{"tool_name":"Bash","session_id":"s"}`,
			wantRole: "",
		},
		{
			name:     "--role overrides the payload, for testing and for a silent harness",
			flag:     "builder",
			payload:  `{"tool_name":"Bash","session_id":"s","agent_type":"reviewer"}`,
			wantRole: "builder",
		},
		{
			name:     "--role fills in when the harness reports no agent",
			flag:     "builder",
			payload:  `{"tool_name":"Bash","session_id":"s"}`,
			wantRole: "builder",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			call, ok := normaliseToolCall("claude", []byte(tc.payload))
			if !ok {
				t.Fatal("payload not understood")
			}
			if got := effectiveRole(tc.flag, call.AgentType); got != tc.wantRole {
				t.Errorf("effectiveRole = %q, want %q", got, tc.wantRole)
			}
		})
	}
}

// TestConsumptionIsScopedToTheActingPersona pins that two personas dispatched in
// ONE session do not share a consumption ledger.
//
// Claude reuses the parent's session_id inside a subagent, so keying the ledger
// by session alone would let the architect's skill runs satisfy the reviewer's
// gate — enforcement that reports success while enforcing nothing, which is the
// exact failure this gate exists to prevent.
func TestConsumptionIsScopedToTheActingPersona(t *testing.T) {
	base := `{"tool_name":"Bash","session_id":"same-session","agent_type":"%s","agent_id":"%s"}`

	reviewer, ok := normaliseToolCall("claude", []byte(fmt.Sprintf(base, "reviewer", "agent-1")))
	if !ok {
		t.Fatal("payload not understood")
	}
	architect, _ := normaliseToolCall("claude", []byte(fmt.Sprintf(base, "architect", "agent-2")))
	mainThread, _ := normaliseToolCall("claude", []byte(`{"tool_name":"Bash","session_id":"same-session"}`))

	if reviewer.ConsumptionScope() == architect.ConsumptionScope() {
		t.Errorf("two personas in one session share a ledger (%q) — one's skill runs would satisfy the other's gate",
			reviewer.ConsumptionScope())
	}
	if mainThread.ConsumptionScope() != "same-session" {
		t.Errorf("a main-thread call must fall back to the session, got %q", mainThread.ConsumptionScope())
	}
	// And the paths they resolve to must differ, which is what actually keeps
	// the ledgers apart on disk.
	dir := t.TempDir()
	if harness.StatePath(dir, reviewer.ConsumptionScope()) == harness.StatePath(dir, architect.ConsumptionScope()) {
		t.Error("distinct scopes resolved to the same state file")
	}
}

// TestConsumptionIsScopedToTheInvocationNotTheRole pins the case the previous
// test does not reach: the SAME persona dispatched twice in one session.
//
// The reviewer on #1410 asked what happens if agent_id is not unique per
// invocation. Separation there is a property of the harness's id, not of this
// code — so what is pinned here is the contract this code owes: given distinct
// ids, two runs of one role must not share a ledger. If the harness ever sends a
// value stable per role, THIS test still passes and the separation is gone,
// which is precisely why the doc comment sends that question to a measurement
// rather than to a unit test.
func TestConsumptionIsScopedToTheInvocationNotTheRole(t *testing.T) {
	base := `{"tool_name":"Bash","session_id":"same-session","agent_type":"reviewer","agent_id":"%s"}`

	first, ok := normaliseToolCall("claude", []byte(fmt.Sprintf(base, "agent-1")))
	if !ok {
		t.Fatal("payload not understood")
	}
	second, _ := normaliseToolCall("claude", []byte(fmt.Sprintf(base, "agent-2")))

	if first.ConsumptionScope() == second.ConsumptionScope() {
		t.Errorf("two dispatches of one role share a ledger (%q) — the second would inherit the first's consumption and go ungated",
			first.ConsumptionScope())
	}
}
