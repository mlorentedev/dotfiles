package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		// dispatched is the session's witnessed dispatches (HARNESS-109). Nil
		// is a session in which the gate saw no Agent call, which is every row
		// that predates that spec.
		dispatched map[string]string
		wantNil    bool
		wantSaid   bool
		// wantRes is the part a caller can act on AFTER the call. The stderr
		// line tells a human reading a live terminal that enforcement is off;
		// this is what lets the decision record say so durably.
		wantRes roleResolution
	}{
		{
			name: "no role asked for is silent — nothing failed to resolve",
			role: "", repoRoot: root, wantNil: true, wantSaid: false, wantRes: roleNotAsked,
		},
		{
			name: "a role that does not exist is allowed AND announced",
			role: "reviewr", repoRoot: root, wantNil: true, wantSaid: true, wantRes: roleUnresolved,
		},
		{
			name: "a repo root without harness/agents is allowed AND announced",
			role: "reviewer", repoRoot: t.TempDir(), wantNil: true, wantSaid: true, wantRes: roleUnresolved,
		},
		{
			name: "a role that resolves is silent",
			role: "reviewer", repoRoot: root, wantNil: false, wantSaid: false, wantRes: roleResolved,
		},
		// HARNESS-109 (#1434). The first row IS the bug: before the dispatch
		// map, a named dispatch of a real persona was the row above it —
		// unresolved, announced, enforcement off — so naming a subagent turned
		// its own gate off. It must now be indistinguishable from an unnamed
		// dispatch of the same persona.
		{
			name: "a NAMED dispatch resolves through the map to its true persona",
			role: "harness109-probe", repoRoot: root,
			dispatched: map[string]string{"harness109-probe": "reviewer"},
			wantNil:    false, wantSaid: false, wantRes: roleResolved,
		},
		{
			name: "a named dispatch of a BUILT-IN agent is quiet, not a fault",
			role: "kubelab-harness", repoRoot: root,
			dispatched: map[string]string{"kubelab-harness": "general-purpose"},
			wantNil:    true, wantSaid: false, wantRes: roleNotAPersona,
		},
		{
			name: "an UNNAMED dispatch of a built-in agent is quiet too",
			role: "general-purpose", repoRoot: root,
			dispatched: map[string]string{"general-purpose": "general-purpose"},
			wantNil:    true, wantSaid: false, wantRes: roleNotAPersona,
		},
		{
			name: "a dispatch the gate never witnessed stays LOUD",
			role: "resumed-from-another-session", repoRoot: root,
			dispatched: map[string]string{"someone-else": "reviewer"},
			wantNil:    true, wantSaid: true, wantRes: roleUnresolved,
		},
		{
			name: "a map entry pointing at a persona whose record is gone is not a false resolve",
			role: "harness109-probe", repoRoot: t.TempDir(),
			dispatched: map[string]string{"harness109-probe": "reviewer"},
			wantNil:    true, wantSaid: false, wantRes: roleNotAPersona,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			got, res := loadGatePersona(&buf, tc.repoRoot, tc.role, tc.dispatched)

			if (got == nil) != tc.wantNil {
				t.Errorf("persona nil = %v, want %v", got == nil, tc.wantNil)
			}
			// Both nil cases return nil and both allow. Only this separates
			// them, and conflating them is what made "enforcement is off" and
			// "nothing to enforce" the same observation.
			if res != tc.wantRes {
				t.Errorf("resolution = %v, want %v", res, tc.wantRes)
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

// runGate drives the real command end to end, in process.
//
// End to end matters here: the three paths that return before any persona is
// loaded are exactly the ones that used to leave no trace, and a test calling
// Decide directly cannot see them at all — which is how they came to be
// unrecorded while every unit test passed.
func runGate(t *testing.T, args []string, payload string) (int, string) {
	t.Helper()
	c := newHarnessGateCmd()
	var errb bytes.Buffer
	c.SetIn(strings.NewReader(payload))
	c.SetOut(io.Discard)
	c.SetErr(&errb)
	c.SetArgs(args)
	return ExitCode(c.Execute()), errb.String()
}

func readJournal(t *testing.T, stateDir, scope string) []harness.DecisionRecord {
	t.Helper()
	recs, err := harness.LoadDecisions(harness.DecisionPath(stateDir, scope))
	if err != nil {
		t.Fatalf("read journal for scope %q: %v", scope, err)
	}
	return recs
}

// blockingRepoRoot writes a persona whose skill is `enforce: block`.
//
// No shipped record declares one — the canary migrated `reviewer` to `warn`
// deliberately — so the block path has to be driven from a fixture. That is the
// honest arrangement rather than a gap: the day a real record turns `block` on,
// this test already covers what happens, and it does not silently start
// measuring something different because the roster changed.
func blockingRepoRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "harness", "agents", "gatekeeper")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := "---\nname: gatekeeper\nkind: persona\nskills:\n  - id: audit\n    enforce: block\n---\n\n# Gatekeeper\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte(rec), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestEveryGateDecisionLeavesADurableRecord is AC5.
//
// The gate's answer to the harness is an exit code, and an exit code is consumed
// and discarded. `warn` and `allow` share it; "the payload did not parse" and
// "there was nothing to enforce" share it; and "the role did not resolve, so
// enforcement is OFF" shares it with a clean pass. Everything that told those
// apart was stderr, which a PreToolUse hook does not persist on exit 0.
//
// So this asserts the property that makes the gate measurable at all: NO path
// returns without writing down what it decided and why.
func TestEveryGateDecisionLeavesADurableRecord(t *testing.T) {
	root := repoRootForTest(t)
	blocking := blockingRepoRoot(t)

	for _, tc := range []struct {
		name        string
		payload     string
		repoRoot    string
		wantScope   string
		wantOutcome harness.Outcome
		wantExit    int
		wantAllowed bool
	}{
		{
			// The single most diagnostic case. An unparsed payload is what a
			// harness with different field names produces — the open question
			// for agy — and before this record it was indistinguishable from a
			// harness working perfectly.
			name:      "a payload the gate cannot parse",
			payload:   `not json at all`,
			repoRoot:  root,
			wantScope: harness.UnparsedScope, wantOutcome: harness.OutcomePayloadUnrecognised,
			wantExit: 0, wantAllowed: true,
		},
		{
			name:      "a skill invocation, which returns before any persona is loaded",
			payload:   `{"tool_name":"Skill","tool_input":{"skill":"audit"},"session_id":"s1","agent_type":"reviewer","agent_id":"a1"}`,
			repoRoot:  root,
			wantScope: "s1-a1", wantOutcome: harness.OutcomeSkillConsumed,
			wantExit: 0, wantAllowed: true,
		},
		{
			name:      "the skill primitive with an unreadable argument",
			payload:   `{"tool_name":"Skill","tool_input":{},"session_id":"s2"}`,
			repoRoot:  root,
			wantScope: "s2", wantOutcome: harness.OutcomeSkillUnnamed,
			wantExit: 0, wantAllowed: true,
		},
		{
			name:      "a main-thread call naming no persona",
			payload:   `{"tool_name":"Bash","session_id":"s3"}`,
			repoRoot:  root,
			wantScope: "s3", wantOutcome: harness.OutcomeNoRole,
			wantExit: 0, wantAllowed: true,
		},
		{
			// AC5 names this case explicitly, and this is why: it ALLOWS, so
			// from outside it is identical to health, while enforcement is off.
			name:      "a role that was asked for and did not resolve",
			payload:   `{"tool_name":"Bash","session_id":"s4","agent_type":"no-such-persona"}`,
			repoRoot:  root,
			wantScope: "s4", wantOutcome: harness.OutcomeRoleUnresolved,
			wantExit: 0, wantAllowed: true,
		},
		{
			name:      "a persona whose skills are all warn",
			payload:   `{"tool_name":"Bash","session_id":"s5","agent_type":"reviewer","agent_id":"a2"}`,
			repoRoot:  root,
			wantScope: "s5-a2", wantOutcome: harness.OutcomeWarn,
			wantExit: 0, wantAllowed: true,
		},
		{
			name:      "a persona whose skills carry no severity is not enforced",
			payload:   `{"tool_name":"Bash","session_id":"s6","agent_type":"builder","agent_id":"a3"}`,
			repoRoot:  root,
			wantScope: "s6-a3", wantOutcome: harness.OutcomeAllow,
			wantExit: 0, wantAllowed: true,
		},
		{
			name:      "a blocked call records before it exits",
			payload:   `{"tool_name":"Bash","session_id":"s7","agent_type":"gatekeeper","agent_id":"a4"}`,
			repoRoot:  blocking,
			wantScope: "s7-a4", wantOutcome: harness.OutcomeBlock,
			wantExit: 2, wantAllowed: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := t.TempDir()
			code, _ := runGate(t, []string{
				"--harness", "claude", "--repo-root", tc.repoRoot, "--state-dir", stateDir,
			}, tc.payload)

			if code != tc.wantExit {
				t.Errorf("exit = %d, want %d", code, tc.wantExit)
			}
			recs := readJournal(t, stateDir, tc.wantScope)
			if len(recs) != 1 {
				t.Fatalf("journal for scope %q holds %d records, want exactly 1 — "+
					"a path that returns without recording is invisible afterwards", tc.wantScope, len(recs))
			}
			got := recs[0]
			if got.Outcome != tc.wantOutcome {
				t.Errorf("outcome = %q, want %q", got.Outcome, tc.wantOutcome)
			}
			if got.Allowed != tc.wantAllowed {
				t.Errorf("allowed = %v, want %v", got.Allowed, tc.wantAllowed)
			}
			if got.Harness != "claude" {
				t.Errorf("harness = %q, want claude", got.Harness)
			}
			if got.Time == "" {
				t.Error("record carries no timestamp")
			}
		})
	}
}

// TestGateRecordCarriesAgentType is AC7, and it is the criterion that converts a
// standing inference into a measurement.
//
// `agent_type` has been DOCUMENTED and acted upon since the binding landed, and
// never observed in any durable artifact — the consumption ledger records only
// the skill name, and the skill path returns before the persona lookup that
// would otherwise have exercised the field. So a dispatch could carry it, or
// not, and both looked the same on disk.
//
// It is read straight off the payload here, before any lookup, so even the two
// early-returning skill paths preserve it.
func TestGateRecordCarriesAgentType(t *testing.T) {
	root := repoRootForTest(t)

	for _, tc := range []struct {
		name    string
		payload string
		scope   string
	}{
		{
			name:    "on the skill path, which returns before the persona is loaded",
			payload: `{"tool_name":"Skill","tool_input":{"skill":"audit"},"session_id":"s","agent_type":"reviewer","agent_id":"a1"}`,
			scope:   "s-a1",
		},
		{
			name:    "on the enforcement path",
			payload: `{"tool_name":"Bash","session_id":"s","agent_type":"reviewer","agent_id":"a1"}`,
			scope:   "s-a1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := t.TempDir()
			runGate(t, []string{"--harness", "claude", "--repo-root", root, "--state-dir", stateDir}, tc.payload)

			recs := readJournal(t, stateDir, tc.scope)
			if len(recs) != 1 {
				t.Fatalf("want 1 record, got %d", len(recs))
			}
			if recs[0].AgentType != "reviewer" {
				t.Errorf("agent_type = %q, want reviewer — the field the whole persona "+
					"resolution rests on is still unrecorded", recs[0].AgentType)
			}
			if recs[0].AgentID != "a1" {
				t.Errorf("agent_id = %q, want a1", recs[0].AgentID)
			}
		})
	}
}

// TestTheRecordSeparatesWhatWasAskedForFromWhatResolved.
//
// Three fields, because they can differ and the difference IS the diagnosis: the
// payload said one thing, `--role` may override it, and the record that actually
// loaded may be a third. Collapsing them into one would hide a `--role` override
// and a renamed record behind the same value.
func TestTheRecordSeparatesWhatWasAskedForFromWhatResolved(t *testing.T) {
	stateDir := t.TempDir()
	runGate(t, []string{
		"--harness", "claude", "--repo-root", repoRootForTest(t), "--state-dir", stateDir,
		"--role", "builder",
	}, `{"tool_name":"Bash","session_id":"s","agent_type":"reviewer","agent_id":"a1"}`)

	recs := readJournal(t, stateDir, "s-a1")
	if len(recs) != 1 {
		t.Fatalf("want 1 record, got %d", len(recs))
	}
	got := recs[0]
	if got.AgentType != "reviewer" {
		t.Errorf("agent_type = %q, want the payload's reviewer", got.AgentType)
	}
	if got.RoleRequested != "builder" {
		t.Errorf("role_requested = %q, want the overriding builder", got.RoleRequested)
	}
	if got.RoleResolved != "builder" {
		t.Errorf("role_resolved = %q, want builder", got.RoleResolved)
	}
}

// TestAWarnIsReadableAfterTheSessionEnds is AC6.
//
// A warn is emitted on stderr, and a PreToolUse hook's stderr on exit 0 is not
// persisted anywhere the session can be asked about afterwards — measured, and
// the subject of lesson 254. So before this, a warn that fired and a warn that
// never fired were the same observation, and the canary accumulated no evidence
// for a whole day while appearing to work.
//
// The journal is read here through a SEPARATE call that shares nothing with the
// writer but the path, which is what "after the session ends" means in practice.
func TestAWarnIsReadableAfterTheSessionEnds(t *testing.T) {
	stateDir := t.TempDir()
	_, stderr := runGate(t, []string{
		"--harness", "claude", "--repo-root", repoRootForTest(t), "--state-dir", stateDir,
	}, `{"tool_name":"Bash","session_id":"s","agent_type":"reviewer","agent_id":"a1"}`)

	// The stderr channel still says it, for a human watching live.
	if !strings.Contains(stderr, "[gate] warn:") {
		t.Errorf("no warn on stderr: %q", stderr)
	}

	recs := readJournal(t, stateDir, "s-a1")
	if len(recs) != 1 {
		t.Fatalf("want 1 record, got %d", len(recs))
	}
	if recs[0].Outcome != harness.OutcomeWarn {
		t.Fatalf("outcome = %q, want warn", recs[0].Outcome)
	}
	// And it names WHICH skills, or the record says a warn happened without
	// saying what would clear it — true, and useless.
	if len(recs[0].Warned) == 0 {
		t.Error("the record reports a warn but names no skill")
	}
	for _, w := range recs[0].Warned {
		if strings.TrimSpace(w) == "" {
			t.Error("a warned skill is blank")
		}
	}
}

// TestTheJournalNeverRecordsToolInput is a security property, not a detail.
//
// This file is durable, unencrypted, world-readable to the user's own processes,
// and nothing scans it. Tool inputs carry file contents, shell commands and
// credentials, so a journal that logged them would be a secrets leak with a
// retention policy — and it would be written by a hook that fires on EVERY tool
// call, which is the worst possible collection rate.
//
// The neutral ToolCall already drops them at the parse boundary. This pins that,
// because the next person adding a field to the record will be looking at a
// payload that still has them.
func TestTheJournalNeverRecordsToolInput(t *testing.T) {
	stateDir := t.TempDir()
	const secret = "AKIAIOSFODNN7EXAMPLE-not-a-real-key"
	payload := `{"tool_name":"Bash","session_id":"s","agent_type":"reviewer","agent_id":"a1",` +
		`"tool_input":{"command":"aws configure set aws_secret_access_key ` + secret + `"}}`

	runGate(t, []string{
		"--harness", "claude", "--repo-root", repoRootForTest(t), "--state-dir", stateDir,
	}, payload)

	raw, err := os.ReadFile(harness.DecisionPath(stateDir, "s-a1"))
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	if strings.Contains(string(raw), secret) {
		t.Fatal("the decision journal contains a tool input — this file is a durable, " +
			"unscanned artifact written on every tool call, so tool arguments must never reach it")
	}
	// The tool's NAME is recorded, and must be: without it the journal cannot
	// say what was gated.
	if !strings.Contains(string(raw), `"tool":"Bash"`) {
		t.Errorf("the tool name is missing from the record: %s", raw)
	}
}

// TestAnUnparsedPayloadRecordsItsSizeNotItsContent.
//
// Size separates "stdin was empty" from "a real payload arrived and did not
// parse" — two causes of a silent gate needing opposite fixes, and the exact
// question agy poses. Content is the one thing that must not be written: the
// payload we could not parse is the payload we cannot vouch for.
func TestAnUnparsedPayloadRecordsItsSizeNotItsContent(t *testing.T) {
	stateDir := t.TempDir()
	const marker = "unparseable-but-sensitive"
	runGate(t, []string{"--harness", "agy", "--state-dir", stateDir}, `{"weird":"`+marker+`"}`)

	recs := readJournal(t, stateDir, harness.UnparsedScope)
	if len(recs) != 1 {
		t.Fatalf("want 1 record, got %d", len(recs))
	}
	if recs[0].PayloadBytes == 0 {
		t.Error("payload_bytes is 0 for a non-empty payload; empty stdin and an " +
			"unrecognised shape would be indistinguishable")
	}
	if recs[0].Harness != "agy" {
		t.Errorf("harness = %q, want agy — an unparsed record that does not say WHICH "+
			"harness produced it cannot be acted on", recs[0].Harness)
	}
	raw, _ := os.ReadFile(harness.DecisionPath(stateDir, harness.UnparsedScope))
	if strings.Contains(string(raw), marker) {
		t.Fatal("the unparsed payload's content was written to disk")
	}
}
