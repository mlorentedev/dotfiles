package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidDispatchNameMatchesTheToolsOwnConstraint(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"reviewer", true},
		{"general-purpose", true},
		{"harness109_probe", true},
		{"a", true},
		{strings.Repeat("a", 64), true},
		{strings.Repeat("a", 65), false}, // one past the tool's own limit
		{"-leading-hyphen", false},       // the pattern requires alphanumeric first
		{"", false},                      // nothing to key by
		{"has space", false},             // free text starts here
		{"../../etc/passwd", false},      // the value lands in a lookup path
		{"name\nwith-newline", false},    // would corrupt any line-oriented reader
		{"café", false},                  // the tool's pattern is ASCII
	} {
		if got := ValidDispatchName(tc.in); got != tc.want {
			t.Errorf("ValidDispatchName(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestRecordDispatchRoundTrips(t *testing.T) {
	path := DispatchPath(t.TempDir(), "s1")

	if err := RecordDispatch(path, "probe", "reviewer"); err != nil {
		t.Fatalf("RecordDispatch: %v", err)
	}
	if got := LoadDispatched(path)["probe"]; got != "reviewer" {
		t.Errorf("probe -> %q, want reviewer", got)
	}

	// An UNNAMED dispatch is keyed by its own type. This is what lets the gate
	// tell "a built-in agent I watched being dispatched" from "a name I have
	// never seen", with no hardcoded list of built-ins anywhere.
	if err := RecordDispatch(path, "", "general-purpose"); err != nil {
		t.Fatalf("RecordDispatch unnamed: %v", err)
	}
	if got := LoadDispatched(path)["general-purpose"]; got != "general-purpose" {
		t.Errorf("general-purpose -> %q, want itself", got)
	}

	// Latest wins: the most recent dispatch is the one whose calls arrive.
	if err := RecordDispatch(path, "probe", "builder"); err != nil {
		t.Fatalf("RecordDispatch rebind: %v", err)
	}
	if got := LoadDispatched(path)["probe"]; got != "builder" {
		t.Errorf("after rebind probe -> %q, want builder", got)
	}
}

func TestRecordDispatchRefusesWhatItCannotSafelyStore(t *testing.T) {
	for _, tc := range []struct{ label, name, agentType string }{
		{"no type to remember", "probe", ""},
		{"name is not an identifier", "../escape", "reviewer"},
		{"type is not an identifier", "probe", "../escape"},
		{"type is free text", "probe", "reviewer; rm -rf /"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			path := DispatchPath(t.TempDir(), "s1")
			if err := RecordDispatch(path, tc.name, tc.agentType); err != nil {
				t.Fatalf("RecordDispatch must not error on a rejected value: %v", err)
			}
			if got := LoadDispatched(path); len(got) != 0 {
				t.Errorf("map = %v, want empty", got)
			}
		})
	}
}

// TestTheDispatchMapIsBounded asserts the cap, because a hook that appends
// without one is a way to fill a disk — and the gate's fail-open discipline
// assumes its own writes are harmless, which stops being true on a full disk.
// The same reasoning behind maxDecisionBytes.
func TestTheDispatchMapIsBounded(t *testing.T) {
	path := DispatchPath(t.TempDir(), "s1")
	for i := 0; i < maxDispatchEntries+50; i++ {
		if err := RecordDispatch(path, fmt.Sprintf("probe-%d", i), "reviewer"); err != nil {
			t.Fatalf("RecordDispatch(%d): %v", i, err)
		}
	}
	got := LoadDispatched(path)
	if len(got) != maxDispatchEntries {
		t.Errorf("map holds %d entries, want the cap of %d", len(got), maxDispatchEntries)
	}
	// Past the cap an existing key is still REBOUND — a long session must not
	// be stuck answering with a stale type for a name it just redispatched.
	if err := RecordDispatch(path, "probe-0", "builder"); err != nil {
		t.Fatalf("rebind past the cap: %v", err)
	}
	if v := LoadDispatched(path)["probe-0"]; v != "builder" {
		t.Errorf("probe-0 -> %q past the cap, want builder", v)
	}
}

// TestALostDispatchMapDegradesToTheOldBehaviour is the fail-open property.
//
// Every failure mode of this file must cost enforcement and never cause a block,
// because the gate's whole contract is that its own malfunction is not a refused
// tool call. Unreadable, corrupt and absent are the three ways it can be lost.
func TestALostDispatchMapDegradesToTheOldBehaviour(t *testing.T) {
	dir := t.TempDir()

	if got := LoadDispatched(DispatchPath(dir, "never-written")); len(got) != 0 {
		t.Errorf("absent map = %v, want empty", got)
	}

	corrupt := DispatchPath(dir, "corrupt")
	if err := os.MkdirAll(filepath.Dir(corrupt), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corrupt, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadDispatched(corrupt); len(got) != 0 {
		t.Errorf("corrupt map = %v, want empty rather than an error", got)
	}

	// A half-written entry — a type that survived a truncated write as an empty
	// string — must not answer a lookup with "".
	partial := DispatchPath(dir, "partial")
	raw, _ := json.Marshal(dispatchState{Types: map[string]string{"probe": "", "": "reviewer"}})
	if err := os.WriteFile(partial, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadDispatched(partial); len(got) != 0 {
		t.Errorf("map with empty halves = %v, want them dropped", got)
	}
}

// TestDispatchPathIsKeyedBySessionNotByScope pins the decision the mechanism
// rests on.
//
// A subagent reuses its PARENT's session id, so the parent writes and the child
// reads the same file. Keying this the way the consumption ledger is keyed —
// session plus agent id — would file the write where the child never looks, and
// the fix would silently do nothing. This test fails if someone "makes it
// consistent" with the ledger.
func TestDispatchPathIsKeyedBySessionNotByScope(t *testing.T) {
	dir := t.TempDir()
	parent := ToolCall{SessionID: "s1"}
	child := ToolCall{SessionID: "s1", AgentID: "aprobe-1"}

	if DispatchPath(dir, parent.SessionID) != DispatchPath(dir, child.SessionID) {
		t.Fatal("parent and child must share one dispatch map")
	}
	if DispatchPath(dir, child.SessionID) == DispatchPath(dir, child.ConsumptionScope()) {
		t.Error("the dispatch map must NOT be keyed by the consumption scope: the child would never find the parent's write")
	}
	// And it does not collide with the ledger or the journal for the same key.
	seen := map[string]bool{
		DispatchPath(dir, "s1"): true,
		StatePath(dir, "s1"):    true,
		DecisionPath(dir, "s1"): true,
	}
	if len(seen) != 3 {
		t.Error("the dispatch map, the ledger and the journal must be three distinct files")
	}
}

// TestAStrandedTempFileIsSwept covers round 2's first finding.
//
// The 5-second hook timeout this spec adds is enforced by KILLING the gate, and
// a killed process runs no deferred cleanup — so every timeout mid-write leaves
// a `.dispatch-*.tmp` behind forever. A leak rather than a fault, which is the
// kind nobody notices until the directory is enormous.
func TestAStrandedTempFileIsSwept(t *testing.T) {
	dir := t.TempDir()
	path := DispatchPath(dir, "s1")
	gateDir := filepath.Dir(path)
	if err := os.MkdirAll(gateDir, 0o750); err != nil {
		t.Fatal(err)
	}

	stranded := filepath.Join(gateDir, ".dispatch-killed.tmp")
	if err := os.WriteFile(stranded, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(stranded, old, old); err != nil {
		t.Fatal(err)
	}

	// A file that could still belong to a gate writing RIGHT NOW must survive:
	// deleting it under that process turns a leak into a lost write.
	live := filepath.Join(gateDir, ".dispatch-inflight.tmp")
	if err := os.WriteFile(live, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	// And nothing else in the directory is this function's business.
	bystander := filepath.Join(gateDir, "s1.decisions.jsonl")
	if err := os.WriteFile(bystander, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(bystander, old, old); err != nil {
		t.Fatal(err)
	}

	if err := RecordDispatch(path, "probe", "reviewer"); err != nil {
		t.Fatalf("RecordDispatch: %v", err)
	}

	if _, err := os.Stat(stranded); !os.IsNotExist(err) {
		t.Errorf("the abandoned temp file survived (err=%v)", err)
	}
	if _, err := os.Stat(live); err != nil {
		t.Errorf("a temp file young enough to be in flight was deleted: %v", err)
	}
	if _, err := os.Stat(bystander); err != nil {
		t.Errorf("the sweep removed a file that is not its business: %v", err)
	}
	if got := LoadDispatched(path)["probe"]; got != "reviewer" {
		t.Errorf("probe -> %q after the sweep, want reviewer", got)
	}
}

// TestLoadDispatchedIsBoundedInMemory covers round 2's second finding.
//
// LoadDispatched runs on EVERY tool call. An unbounded read turns a stray large
// file in the state dir into an OOM in every session — the one way a gate whose
// whole contract is fail-open can still take a session down.
func TestLoadDispatchedIsBoundedInMemory(t *testing.T) {
	dir := t.TempDir()
	path := DispatchPath(dir, "s1")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	// Valid JSON far larger than the bound: without the limit this parses and
	// returns entries, so the assertion below distinguishes "bounded" from
	// "happened not to read it".
	var b strings.Builder
	b.WriteString(`{"types":{`)
	for i := 0; b.Len() < maxDispatchBytes*2; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `"probe-%d":"reviewer"`, i)
	}
	b.WriteString(`}}`)
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() <= maxDispatchBytes {
		t.Fatalf("fixture is not over the bound (size check err=%v)", err)
	}

	// Truncated at the limit, so it no longer parses, so it degrades to empty —
	// the documented behaviour for anything unreadable.
	if got := LoadDispatched(path); len(got) != 0 {
		t.Errorf("read %d entries from an oversized file, want the bound to truncate it into an empty degrade", len(got))
	}
}
