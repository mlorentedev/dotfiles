package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// HARNESS-109 (#1434) end to end.
//
// EVERY TEST HERE DRIVES THE REAL COMMAND, for the reason runGate documents: the
// bug being fixed lives in the wiring between three functions that are each
// correct alone. `loadGatePersona` was already right about what it was asked;
// `normaliseToolCall` was already right about what arrived; nothing joined the
// parent's dispatch to the child's call, and no unit test of either could see
// that, which is why the defect survived until the decision journal caught it.

// dispatchPayload is the PARENT's call: the Agent tool, with the arguments that
// say who is being dispatched.
func dispatchPayload(session, name, agentType string) string {
	input := map[string]any{"prompt": "do the thing", "description": "a task"}
	if name != "" {
		input["name"] = name
	}
	if agentType != "" {
		input["subagent_type"] = agentType
	}
	raw, err := json.Marshal(map[string]any{
		"session_id": session,
		"tool_name":  "Agent",
		"tool_input": input,
	})
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// childPayload is a tool call made INSIDE the dispatched subagent. agent_type
// carries whatever the harness put there — the name, when the dispatch was
// named, which is the whole defect.
func childPayload(session, agentType, agentID string) string {
	return fmt.Sprintf(`{"tool_name":"Bash","session_id":%q,"agent_type":%q,"agent_id":%q}`,
		session, agentType, agentID)
}

func lastRecord(t *testing.T, stateDir, scope string) harness.DecisionRecord {
	t.Helper()
	recs := readJournal(t, stateDir, scope)
	if len(recs) == 0 {
		t.Fatalf("no decision recorded for scope %q", scope)
	}
	return recs[len(recs)-1]
}

// TestANamedDispatchIsGatedLikeAnUnnamedOne is the acceptance criterion for the
// whole spec, written as the comparison #1434's own table makes: two dispatches
// of the SAME persona that differ only in whether the caller supplied a name
// must be gated identically.
//
// Before this change the named row recorded `role-unresolved` and enforcement
// was off for every call the subagent made. The test asserts the two rows agree
// on the resolved role, not merely that the named one is non-empty: "resolves to
// something" would also pass if it resolved to the wrong persona.
func TestANamedDispatchIsGatedLikeAnUnnamedOne(t *testing.T) {
	root := repoRootForTest(t)

	resolvedFor := func(t *testing.T, name string) harness.DecisionRecord {
		t.Helper()
		stateDir := t.TempDir()
		session := "s-" + strings.ReplaceAll(name, "_", "-")
		args := []string{"--harness", "claude", "--repo-root", root, "--state-dir", stateDir}

		if code, _ := runGate(t, args, dispatchPayload(session, name, "reviewer")); code != 0 {
			t.Fatalf("dispatch call exited %d, want 0", code)
		}
		// agent_type carries the NAME when one was given — the harness's
		// behaviour, measured, not a choice this test makes.
		acting := name
		if acting == "" {
			acting = "reviewer"
		}
		if code, _ := runGate(t, args, childPayload(session, acting, "a"+acting+"-abc")); code != 0 {
			t.Fatalf("child call exited %d, want 0", code)
		}
		return lastRecord(t, stateDir, session+"-a"+acting+"-abc")
	}

	unnamed := resolvedFor(t, "")
	named := resolvedFor(t, "harness109-probe")

	if unnamed.RoleResolved != "reviewer" {
		t.Fatalf("unnamed dispatch resolved %q, want reviewer — the control is broken, not the fix", unnamed.RoleResolved)
	}
	if named.RoleResolved != unnamed.RoleResolved {
		t.Errorf("named dispatch resolved %q, unnamed resolved %q — naming a subagent still changes its gate",
			named.RoleResolved, unnamed.RoleResolved)
	}
	if named.Outcome != unnamed.Outcome {
		t.Errorf("named outcome = %q, unnamed = %q — the two must be indistinguishable", named.Outcome, unnamed.Outcome)
	}
	// The WARNED LIST, not just the outcome. AC2 claims the resolved persona's
	// declared enforce modes are applied, and `warn` == `warn` would hold even if
	// the two rows warned about different skills. Raised by the independent
	// review on #1471: until this line the equality was proven only by the live
	// AC6 evidence, which no test re-runs.
	if !slices.Equal(named.Warned, unnamed.Warned) {
		t.Errorf("named warned %v, unnamed warned %v — the same persona must produce the same forced skills",
			named.Warned, unnamed.Warned)
	}
	if len(named.Warned) == 0 {
		t.Error("no skills were warned about, so this test would pass against a persona that enforces nothing")
	}
	// The raw name is kept, so the journal can still say WHICH dispatch this
	// was. Resolving must not erase the caller's own identifier.
	if named.RoleRequested != "harness109-probe" {
		t.Errorf("role_requested = %q, want the raw dispatch name", named.RoleRequested)
	}
	if named.AgentType != "harness109-probe" {
		t.Errorf("agent_type = %q, want what the payload actually said", named.AgentType)
	}
}

// TestWithoutTheDispatchTheChildStillFailsOpen is the control the test above
// needs to mean anything.
//
// If a named child resolved even with NO preceding dispatch, the map would not
// be what is doing the work and the test above would pass for the wrong reason —
// the shape of finding that the review on #1455 caught three times.
func TestWithoutTheDispatchTheChildStillFailsOpen(t *testing.T) {
	root := repoRootForTest(t)
	stateDir := t.TempDir()
	args := []string{"--harness", "claude", "--repo-root", root, "--state-dir", stateDir}

	code, stderr := runGate(t, args, childPayload("s-orphan", "harness109-probe", "aprobe-1"))
	if code != 0 {
		t.Fatalf("exit = %d, want 0: an unresolved role must never block", code)
	}
	rec := lastRecord(t, stateDir, "s-orphan-aprobe-1")
	if rec.Outcome != harness.OutcomeRoleUnresolved {
		t.Errorf("outcome = %q, want %q for a dispatch the gate never witnessed", rec.Outcome, harness.OutcomeRoleUnresolved)
	}
	if !strings.Contains(stderr, "ENFORCEMENT IS OFF") {
		t.Errorf("a genuinely unknown role must stay loud; stderr = %q", stderr)
	}
}

// TestAWitnessedBuiltInAgentIsQuietNotAFault covers the half of the fix that is
// not about resolving anything.
//
// `general-purpose`, `Explore` and `Plan` have no persona BY DESIGN. Reporting
// them as `role-unresolved` with "ENFORCEMENT IS OFF" was a misclassification,
// and it is why 271 of 274 records read as faults: the genuine faults were
// unfindable among them. Both routes to a built-in are covered — named and
// unnamed — because they take different branches.
func TestAWitnessedBuiltInAgentIsQuietNotAFault(t *testing.T) {
	root := repoRootForTest(t)

	for _, tc := range []struct{ name, dispatchName, acting string }{
		{"unnamed built-in", "", "general-purpose"},
		{"named built-in", "kubelab-harness", "kubelab-harness"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := t.TempDir()
			session := "s-builtin"
			args := []string{"--harness", "claude", "--repo-root", root, "--state-dir", stateDir}

			if code, _ := runGate(t, args, dispatchPayload(session, tc.dispatchName, "general-purpose")); code != 0 {
				t.Fatalf("dispatch exited %d", code)
			}
			code, stderr := runGate(t, args, childPayload(session, tc.acting, "a"+tc.acting+"-1"))
			if code != 0 {
				t.Fatalf("child exited %d, want 0", code)
			}
			rec := lastRecord(t, stateDir, session+"-a"+tc.acting+"-1")
			if rec.Outcome != harness.OutcomeNoRole {
				t.Errorf("outcome = %q, want %q: a built-in agent is nobody, not a broken somebody",
					rec.Outcome, harness.OutcomeNoRole)
			}
			if strings.Contains(stderr, "ENFORCEMENT IS OFF") {
				t.Errorf("a built-in agent must not be announced as a failure; stderr = %q", stderr)
			}
			// Still distinguishable from a main-thread call, which is the
			// reason no ninth Outcome was added.
			if rec.RoleRequested != tc.acting {
				t.Errorf("role_requested = %q, want %q so the journal still separates this from the main thread",
					rec.RoleRequested, tc.acting)
			}
		})
	}
}

// TestTheDispatchMapNeverStoresAnythingButTwoIdentifiers is the security
// property, asserted on the bytes rather than on the intent.
//
// The gate reads tool input on exactly two paths now, and this one is new, so
// the assertion is that the FILE contains the two identifiers and none of the
// prompt — not that the code meant to write only those.
func TestTheDispatchMapNeverStoresAnythingButTwoIdentifiers(t *testing.T) {
	root := repoRootForTest(t)
	stateDir := t.TempDir()
	args := []string{"--harness", "claude", "--repo-root", root, "--state-dir", stateDir}

	secret := "ghp_thisMustNeverBeWrittenAnywhere"
	payload := fmt.Sprintf(
		`{"session_id":"s-sec","tool_name":"Agent","tool_input":{"name":"probe-1","subagent_type":"reviewer","prompt":%q}}`,
		secret)
	if code, _ := runGate(t, args, payload); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}

	raw, err := os.ReadFile(harness.DispatchPath(stateDir, "s-sec"))
	if err != nil {
		t.Fatalf("read dispatch map: %v", err)
	}
	if strings.Contains(string(raw), secret) {
		t.Fatalf("the dispatch map wrote tool input to disk: %s", raw)
	}
	var got map[string]map[string]string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("dispatch map is not the expected shape: %v", err)
	}
	if len(got["types"]) != 1 || got["types"]["probe-1"] != "reviewer" {
		t.Errorf("map = %v, want exactly one probe-1 -> reviewer entry", got["types"])
	}

	// And the journal is unchanged in this respect: it still carries no tool
	// input at all, dispatch arguments included.
	for _, rec := range readJournal(t, stateDir, "s-sec") {
		blob, _ := json.Marshal(rec)
		if strings.Contains(string(blob), secret) || strings.Contains(string(blob), "probe-1") {
			t.Errorf("the decision journal recorded dispatch arguments: %s", blob)
		}
	}
}

// TestAnUnusableDispatchNameIsNotWritten covers the validation boundary.
//
// A value failing the Agent tool's own name pattern is not written, and the call
// still allows — the containment that makes reading tool input here safe. A
// blocked call would be the far worse outcome, so it is asserted too.
func TestAnUnusableDispatchNameIsNotWritten(t *testing.T) {
	root := repoRootForTest(t)

	for _, tc := range []struct{ label, name, agentType string }{
		{"name with a path separator", "../../etc/passwd", "reviewer"},
		{"name with a space", "my agent", "reviewer"},
		{"type with a path separator", "probe", "../reviewer"},
		{"empty type", "probe", ""},
	} {
		t.Run(tc.label, func(t *testing.T) {
			stateDir := t.TempDir()
			args := []string{"--harness", "claude", "--repo-root", root, "--state-dir", stateDir}

			code, _ := runGate(t, args, dispatchPayload("s-bad", tc.name, tc.agentType))
			if code != 0 {
				t.Fatalf("exit = %d, want 0: a rejected dispatch name must never block", code)
			}
			if got := harness.LoadDispatched(harness.DispatchPath(stateDir, "s-bad")); len(got) != 0 {
				t.Errorf("map = %v, want empty for an unusable dispatch", got)
			}
		})
	}
}

// TestOnlyTheDispatchPrimitiveIsReadForDispatchArguments mirrors
// TestSkillArgOnlyReadsTheSkillPrimitive: a `name` argument on some unrelated
// tool must not be mistaken for a dispatch, or any tool with a `name` parameter
// would start writing map entries.
func TestOnlyTheDispatchPrimitiveIsReadForDispatchArguments(t *testing.T) {
	for _, tc := range []struct {
		tool     string
		wantName string
		wantType string
	}{
		{"Agent", "probe", "reviewer"},
		{"agent", "probe", "reviewer"},
		{"Task", "probe", "reviewer"},
		{"Write", "", ""},
		{"Bash", "", ""},
		{"Skill", "", ""},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			name, agentType := dispatchArgs(tc.tool, map[string]any{
				"name": "probe", "subagent_type": "reviewer",
			})
			if name != tc.wantName || agentType != tc.wantType {
				t.Errorf("dispatchArgs(%q) = (%q, %q), want (%q, %q)",
					tc.tool, name, agentType, tc.wantName, tc.wantType)
			}
		})
	}
}

// brokenRecordRoot writes a persona DIRECTORY whose AGENT.md will not parse.
//
// This is the state the independent review's second finding is about, and it
// cannot be produced from the real roster: a bad merge, a truncated write, a
// half-applied patch. The directory exists — so the name IS a persona — and the
// file does not load, which is a fault and must stay loud rather than be
// mistaken for a built-in agent that has no record by design.
func brokenRecordRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "harness", "agents", "reviewer")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Frontmatter that opens and never closes: the shape a truncated write
	// leaves behind, and one LoadPersona rejects rather than reads as empty.
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"),
		[]byte("---\nname: reviewer\nskills: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestADispatchNamedAfterAnotherPersonaResolvesToItsTrueType is the independent
// review's F1, end to end, as it reproduced it.
//
// A dispatch NAME is unconstrained and may happen to equal a persona's. With the
// roster consulted first, `Agent(name: "reviewer", subagent_type: "builder")`
// resolved to `reviewer` and enforced reviewer's skills on a builder — and the
// journal read `outcome: warn` with a resolved role, which is what health looks
// like. Nothing in the 274 observed records shadows a persona, so this would
// have been found late, under `enforce: block`, as enforcement-by-the-wrong-
// persona selected by one word in a caller's argument.
func TestADispatchNamedAfterAnotherPersonaResolvesToItsTrueType(t *testing.T) {
	root := repoRootForTest(t)
	stateDir := t.TempDir()
	session := "s-shadow"
	args := []string{"--harness", "claude", "--repo-root", root, "--state-dir", stateDir}

	if code, _ := runGate(t, args, dispatchPayload(session, "reviewer", "builder")); code != 0 {
		t.Fatalf("dispatch exited %d, want 0", code)
	}
	if code, _ := runGate(t, args, childPayload(session, "reviewer", "areviewer-shadow")); code != 0 {
		t.Fatalf("child exited %d, want 0", code)
	}

	rec := lastRecord(t, stateDir, session+"-areviewer-shadow")
	if rec.RoleResolved != "builder" {
		t.Errorf("role_resolved = %q, want builder: the map records what the caller DECLARED, the name only what it was CALLED",
			rec.RoleResolved)
	}
	// The shadowed name is still what the payload said, so the journal can show
	// the shadowing rather than hide it behind the corrected answer.
	if rec.RoleRequested != "reviewer" || rec.AgentType != "reviewer" {
		t.Errorf("role_requested/agent_type = %q/%q, want the raw shadowing name in both",
			rec.RoleRequested, rec.AgentType)
	}
}

// TestAnOperatorsRoleFlagOutranksTheDispatchMap pins the limit of the fix above.
//
// The map corrects a PAYLOAD string. `--role` is a human's explicit instruction
// and the map is not about it: a blunt map-first reorder would let a session's
// dispatch history quietly override the operator's own override, which is the
// bug in the opposite direction.
func TestAnOperatorsRoleFlagOutranksTheDispatchMap(t *testing.T) {
	root := repoRootForTest(t)
	stateDir := t.TempDir()
	session := "s-flag"
	args := []string{"--harness", "claude", "--repo-root", root, "--state-dir", stateDir}

	if code, _ := runGate(t, args, dispatchPayload(session, "reviewer", "builder")); code != 0 {
		t.Fatalf("dispatch exited %d", code)
	}
	withRole := append(append([]string{}, args...), "--role", "reviewer")
	if code, _ := runGate(t, withRole, childPayload(session, "reviewer", "aflag-1")); code != 0 {
		t.Fatalf("child exited %d", code)
	}
	rec := lastRecord(t, stateDir, session+"-aflag-1")
	if rec.RoleResolved != "reviewer" {
		t.Errorf("role_resolved = %q, want reviewer: --role must win over the map", rec.RoleResolved)
	}
}

// TestABrokenPersonaRecordStaysLoud is the review's F2, end to end.
//
// Concluding "not a persona" from a failed load conflates `general-purpose`,
// which has no record BY DESIGN, with `reviewer` whose AGENT.md a bad merge just
// broke. Quieting the second hides enforcement being off inside the very cleanup
// that exists to surface it — the defect this spec fixes, committed by the fix.
func TestABrokenPersonaRecordStaysLoud(t *testing.T) {
	root := brokenRecordRoot(t)
	stateDir := t.TempDir()
	session := "s-broken"
	args := []string{"--harness", "claude", "--repo-root", root, "--state-dir", stateDir}

	if code, _ := runGate(t, args, dispatchPayload(session, "probe", "reviewer")); code != 0 {
		t.Fatalf("dispatch exited %d", code)
	}
	code, stderr := runGate(t, args, childPayload(session, "probe", "abroken-1"))
	if code != 0 {
		t.Fatalf("child exited %d, want 0: a broken record must never block", code)
	}
	rec := lastRecord(t, stateDir, session+"-abroken-1")
	if rec.Outcome != harness.OutcomeRoleUnresolved {
		t.Errorf("outcome = %q, want %q: a record that exists and will not load is a FAULT, not a built-in",
			rec.Outcome, harness.OutcomeRoleUnresolved)
	}
	if !strings.Contains(stderr, "ENFORCEMENT IS OFF") {
		t.Errorf("a broken persona record must be announced; stderr = %q", stderr)
	}
}
