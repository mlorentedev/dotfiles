package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestDecisionRecordSchemaIsPinned fixes the wire field names.
//
// This record is an INSTRUMENT, not a log. `features.json` selects on it, the
// agy/pi/opencode verification will select on it, and a human answering "did the
// gate ever see agent_type" greps it. A renamed key breaks every one of those
// silently — the query returns nothing, and nothing is exactly what a working
// system that has not fired yet also returns. Same class as
// `check-roster-consistency.py`'s regex, which returned "no skills" in silence
// when the schema moved under it.
//
// So the names are pinned here, and renaming one is a deliberate act that also
// updates its consumers, rather than a refactor that quietly empties them.
func TestDecisionRecordSchemaIsPinned(t *testing.T) {
	full := DecisionRecord{
		Time: "2026-09-02T00:00:00Z", Harness: "claude", Session: "s", AgentType: "reviewer",
		AgentID: "a1", Scope: "s-a1", Tool: "Bash", Skill: "audit", RoleRequested: "reviewer",
		RoleResolved: "reviewer", Outcome: OutcomeWarn, Allowed: true, Reason: "r",
		Warned: []string{"w"}, Missing: []string{"m"}, PayloadBytes: 7,
	}
	raw, err := json.Marshal(full)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := []string{
		"agent_id", "agent_type", "allowed", "harness", "missing", "payload_bytes",
		"reason", "role_requested", "role_resolved", "scope", "session", "skill",
		"tool", "ts", "warned", "outcome",
	}
	sort.Strings(want)
	var keys []string
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Errorf("the record's wire schema changed.\n got: %v\nwant: %v\n\n"+
			"Every consumer selects on these names. Update them in the same change, or a\n"+
			"query that used to answer now returns nothing and reads as 'never happened'.",
			keys, want)
	}
}

// TestDecisionOutcomesAreDistinct guards the vocabulary itself. Two constants
// sharing a string would silently merge two states — and the pair that matters,
// no-role and role-unresolved, are the ones a reader most needs told apart:
// both allow, and only one means enforcement was off.
func TestDecisionOutcomesAreDistinct(t *testing.T) {
	all := []Outcome{
		OutcomePayloadUnrecognised, OutcomeSkillConsumed, OutcomeSkillUnnamed,
		OutcomeNoRole, OutcomeRoleUnresolved, OutcomeAllow, OutcomeWarn, OutcomeBlock,
	}
	seen := map[Outcome]bool{}
	for _, o := range all {
		if o == "" {
			t.Error("an outcome is the empty string, which is indistinguishable from an unset field")
		}
		if seen[o] {
			t.Errorf("duplicate outcome %q — two states would be recorded as one", o)
		}
		seen[o] = true
	}
}

// TestDecisionPathSharesTheLedgersCollisionGuard is why DecisionPath does not
// build its own filename.
//
// Character-mapping alone collides: `a/b` and `a.b` both flatten to `a_b`. On
// the consumption ledger that opens one session's gate with another's record; on
// the journal it attributes one dispatch's decisions to another, which is worse
// in the specific way that matters — this file is the measurement, so a
// collision corrupts the answer instead of announcing itself.
func TestDecisionPathSharesTheLedgersCollisionGuard(t *testing.T) {
	dir := t.TempDir()
	a, b := "a/b", "a.b"
	if DecisionPath(dir, a) == DecisionPath(dir, b) {
		t.Errorf("scopes %q and %q share a journal at %s", a, b, DecisionPath(dir, a))
	}
	// And the two files for ONE scope must differ, or the journal would append
	// into the ledger and destroy it.
	if DecisionPath(dir, a) == StatePath(dir, a) {
		t.Error("the journal and the consumption ledger resolve to the same file")
	}
	// They must nonetheless agree on the scope's identity, which is the whole
	// reason scopeKey is shared rather than reimplemented.
	stem := strings.TrimSuffix(filepath.Base(StatePath(dir, a)), ".json")
	if !strings.HasPrefix(filepath.Base(DecisionPath(dir, a)), stem) {
		t.Errorf("journal %q is not keyed by the ledger's stem %q",
			filepath.Base(DecisionPath(dir, a)), stem)
	}
}

func TestDecisionJournalRoundTrips(t *testing.T) {
	path := DecisionPath(t.TempDir(), "s")
	for _, o := range []Outcome{OutcomeAllow, OutcomeWarn, OutcomeBlock} {
		if err := RecordDecision(path, DecisionRecord{Outcome: o, Tool: "Bash"}); err != nil {
			t.Fatalf("record %s: %v", o, err)
		}
	}
	got, err := LoadDecisions(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("read %d records, want 3", len(got))
	}
	if got[0].Outcome != OutcomeAllow || got[2].Outcome != OutcomeBlock {
		t.Errorf("records are not oldest-first: %v", got)
	}
	if got[0].Time == "" {
		t.Error("a record was written with no timestamp; ordering across files would be unrecoverable")
	}
}

// TestDecisionJournalSurvivesATornFinalLine pins the read side against the way
// this file actually gets damaged.
//
// The writer is a hook. A hook is killed when the session is, so a half-written
// final line is an expected state rather than corruption. Refusing the whole
// file for it would discard every intact record before it — turning a partial
// answer into no answer, which is the direction this whole spec argues against.
func TestDecisionJournalSurvivesATornFinalLine(t *testing.T) {
	path := DecisionPath(t.TempDir(), "s")
	if err := RecordDecision(path, DecisionRecord{Outcome: OutcomeAllow}); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"outcome":"bl`); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	got, err := LoadDecisions(path)
	if err != nil {
		t.Fatalf("a torn line made the whole journal unreadable: %v", err)
	}
	if len(got) != 1 || got[0].Outcome != OutcomeAllow {
		t.Errorf("the intact record before the tear was lost: %v", got)
	}
}

// TestDecisionJournalIsBounded. The gate runs on EVERY tool call, so an
// unbounded journal is a way to fill a disk — and a full disk breaks the writes
// that the fail-open discipline assumes are harmless to lose.
func TestDecisionJournalIsBounded(t *testing.T) {
	path := DecisionPath(t.TempDir(), "s")
	big := DecisionRecord{Outcome: OutcomeAllow, Reason: strings.Repeat("x", 4096)}
	for i := 0; i < 400; i++ {
		if err := RecordDecision(path, big); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() >= maxDecisionBytes {
		t.Errorf("journal reached %d bytes without rotating (cap %d)", fi.Size(), maxDecisionBytes)
	}
	// Rotation keeps exactly one generation, so the total is bounded rather
	// than merely the live file.
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf("no rotated generation beside the live journal: %v", err)
	}
}

func TestLoadDecisionsOnAMissingJournalIsEmptyNotAnError(t *testing.T) {
	got, err := LoadDecisions(DecisionPath(t.TempDir(), "never-written"))
	if err != nil {
		t.Errorf("a missing journal must read as empty, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want no records, got %d", len(got))
	}
}
