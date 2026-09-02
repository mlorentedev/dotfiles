package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

// TestConcurrentWritersLoseNoRecords answers the reviewer's concern on #1435
// with a measurement rather than with prose about O_APPEND.
//
// The gate runs on every tool call, and several sessions and subagents on one
// machine write at once — so this is the normal case, not an edge one. Each
// writer opens its own descriptor, which is what a separate process does too;
// the kernel path exercised is the same one, so this models cross-process
// appends faithfully for the property being asserted.
//
// The property: every record survives, and none is torn by another writer.
func TestConcurrentWritersLoseNoRecords(t *testing.T) {
	path := DecisionPath(t.TempDir(), "busy")
	const writers, each = 24, 40

	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < each; i++ {
				if err := RecordDecision(path, DecisionRecord{
					Outcome: OutcomeAllow,
					Tool:    "Bash",
					Session: fmt.Sprintf("w%d-i%d", w, i),
				}); err != nil {
					t.Errorf("writer %d record %d: %v", w, i, err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	recs, err := LoadDecisions(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(recs) != writers*each {
		t.Errorf("read %d records, want %d — concurrent appends lost or tore records",
			len(recs), writers*each)
	}
	// Every session id must appear exactly once. A count alone would pass if one
	// record were duplicated while another was lost.
	seen := map[string]int{}
	for _, r := range recs {
		seen[r.Session]++
	}
	for w := 0; w < writers; w++ {
		for i := 0; i < each; i++ {
			if k := fmt.Sprintf("w%d-i%d", w, i); seen[k] != 1 {
				t.Fatalf("record %s appears %d times, want exactly 1", k, seen[k])
			}
		}
	}
}

// TestTheRecordThatTriggersRotationIsInTheRotatedFile pins the ORDERING that
// the reviewer's rotation finding on #1435 is about, deterministically.
//
// Two earlier attempts at this were worse and are worth recording, because both
// failure modes are ones this repository keeps meeting:
//
//   - The first was single-threaded and asserted "no record is lost". The old
//     ordering passes that: stat sees a full file, renames it, writes to a fresh
//     one, nothing lost. It stayed green with the defect reinstated — VACUOUS,
//     and only exposed by proving the red direction.
//   - The second asserted the emergent property under concurrency (the kept
//     generation is always full). That one FLAKED: it depends on goroutine
//     interleaving, and the residual race documented in RecordDecision means it
//     is not a property the code guarantees.
//
// So this asserts the ordering itself, which is what actually changed and is
// fully deterministic. Write-then-rotate puts the triggering record in the
// ROTATED file. Rotate-then-write puts it in a fresh live file and leaves the
// rotated one without it.
func TestTheRecordThatTriggersRotationIsInTheRotatedFile(t *testing.T) {
	path := DecisionPath(t.TempDir(), "s")
	big := strings.Repeat("x", 4096)

	trigger := ""
	for i := 0; i < 600; i++ {
		id := fmt.Sprintf("r%d", i)
		if err := RecordDecision(path, DecisionRecord{
			Outcome: OutcomeAllow, Reason: big, Session: id,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path + ".1"); err == nil {
			trigger = id
			break
		}
	}
	if trigger == "" {
		t.Fatal("no rotation happened, so this test asserts nothing")
	}

	rotated, err := LoadDecisions(path + ".1")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rotated {
		if r.Session == trigger {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("the record that triggered rotation (%s) is not in the rotated file. "+
			"That means rotation ran BEFORE the write, which is the ordering that drops a "+
			"record when another writer rotates in between", trigger)
	}
}
