package env

import (
	"errors"
	"reflect"
	"testing"
)

// CLI-065 (#1363): the sweep. Every test here runs against the fake; the
// registry store's one behaviour the fake must mirror — Delete of an absent
// name succeeds — is pinned on the fake below and stated on the interface.

// AC1: the first run writes the marker naming what it persisted; the second
// run changes nothing, marker included.
func TestPersist_WritesTheMarkerOnce(t *testing.T) {
	vars := []ResolvedVar{{Name: "B", Value: "2"}, {Name: "A", Value: "1"}}
	store := &fakeUserEnv{values: map[string]string{}}

	if _, err := Persist(vars, store); err != nil {
		t.Fatal(err)
	}
	if got := store.values[ManagedMarker]; got != "A;B" {
		t.Fatalf("marker must list the persisted names sorted, got %q", got)
	}

	store.writes, store.ops = nil, nil
	res, err := Persist(vars, store)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res {
		if r.Changed || r.Removed {
			t.Fatalf("second run must change nothing, got %+v", r)
		}
	}
	if len(store.ops) != 0 {
		t.Fatalf("second run must not touch the store, got %v", store.ops)
	}
}

// AC2: a name the contract retires is deleted on the next run — and ONLY it.
// A foreign name present throughout is never touched, because the sweep is
// bounded by the marker, not by what the store holds.
func TestPersist_SweepsOnlyWhatTheMarkerOwns(t *testing.T) {
	store := &fakeUserEnv{values: map[string]string{"FOREIGN": "theirs"}}
	run1 := []ResolvedVar{{Name: "A", Value: "1"}, {Name: "B", Value: "2"}}
	if _, err := Persist(run1, store); err != nil {
		t.Fatal(err)
	}

	run2 := []ResolvedVar{{Name: "A", Value: "1"}} // B retired
	store.ops = nil
	res, err := Persist(run2, store)
	if err != nil {
		t.Fatal(err)
	}
	var removed []string
	for _, r := range res {
		if r.Removed {
			removed = append(removed, r.Name)
		}
	}
	if !reflect.DeepEqual(removed, []string{"B"}) {
		t.Fatalf("exactly B must be removed, got %v", removed)
	}
	if _, ok := store.values["B"]; ok {
		t.Fatal("B still in the store after the sweep")
	}
	if store.values["FOREIGN"] != "theirs" {
		t.Fatalf("a name dotf never wrote was touched: %q", store.values["FOREIGN"])
	}
	if got := store.values[ManagedMarker]; got != "A" {
		t.Fatalf("marker must be rewritten from the contract, got %q", got)
	}

	// Third run: the marker no longer names B, so nothing is left to sweep.
	store.ops = nil
	if _, err := Persist(run2, store); err != nil {
		t.Fatal(err)
	}
	if len(store.ops) != 0 {
		t.Fatalf("a swept store must be stable, got %v", store.ops)
	}
}

// The order is load-bearing (registry names are case-insensitive): on a
// case-only rename the old spelling is a leftover only if compared exactly,
// and write-then-delete would delete the value just written. Two guards: the
// comparison is case-insensitive, so a case-only rename is not a leftover at
// all; and every delete precedes every set, so a future exact comparison
// could not lose the write either.
func TestPersist_DeletesBeforeItWrites(t *testing.T) {
	store := &fakeUserEnv{values: map[string]string{
		"OLD": "x", "Foo": "1", ManagedMarker: "Foo;OLD",
	}}
	vars := []ResolvedVar{{Name: "FOO", Value: "1"}, {Name: "NEW", Value: "2"}}
	if _, err := Persist(vars, store); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.deletes, []string{"OLD"}) {
		t.Fatalf("a case-only rename must not be swept; deletes = %v", store.deletes)
	}
	lastDel, firstSet := -1, len(store.ops)
	for i, op := range store.ops {
		switch op[:4] {
		case "del:":
			lastDel = i
		case "set:":
			if i < firstSet {
				firstSet = i
			}
		}
	}
	if lastDel > firstSet {
		t.Fatalf("every delete must precede every set, got %v", store.ops)
	}
}

// AC5: no marker means no record of what dotf wrote, so nothing is deleted —
// the registry may hold names an old contract persisted, and guessing would be
// the unbounded sweep this exists to avoid. The marker is written so the next
// run has its record.
func TestPersist_NoMarkerDeletesNothing(t *testing.T) {
	store := &fakeUserEnv{values: map[string]string{"STALE_FROM_2026_05": "old", "A": "1"}}
	res, err := Persist([]ResolvedVar{{Name: "A", Value: "1"}}, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.deletes) != 0 {
		t.Fatalf("nothing may be deleted without a marker, got %v", store.deletes)
	}
	if store.values[ManagedMarker] != "A" {
		t.Fatalf("marker must be written on the first run, got %q", store.values[ManagedMarker])
	}
	for _, r := range res {
		if r.Name == ManagedMarker && !r.Changed {
			t.Fatal("the first marker write must be reported as a change")
		}
	}
}

// Leftovers is the one definition three callers share. Its edges: empty and
// missing markers are empty sets (not [""]), the marker never names itself,
// comparison is case-insensitive, duplicates collapse, output is sorted.
func TestLeftovers(t *testing.T) {
	vars := []ResolvedVar{{Name: "A", Value: "1"}, {Name: "Foo", Value: "2"}}
	cases := []struct {
		name   string
		marker []string
		want   []string
	}{
		{"nil marker", nil, nil},
		{"empty string parses to nothing", ParseMarker(""), nil},
		{"all still named", []string{"A", "Foo"}, nil},
		{"one retired", []string{"A", "B", "Foo"}, []string{"B"}},
		{"case-only rename is not a leftover", []string{"FOO", "A"}, nil},
		{"marker never names itself", []string{ManagedMarker, "A"}, nil},
		{"duplicates collapse, output sorted", []string{"Z", "B", "z", "B"}, []string{"B", "Z"}},
		{"blank entries ignored", ParseMarker("A;;B; "), []string{"B"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Leftovers(tc.marker, vars); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Leftovers(%v) = %v, want %v", tc.marker, got, tc.want)
			}
		})
	}
}

// MarkerValue is deterministic: order-independent, deduplicated, sorted.
func TestMarkerValue(t *testing.T) {
	a := MarkerValue([]ResolvedVar{{Name: "B"}, {Name: "A"}, {Name: "B"}})
	b := MarkerValue([]ResolvedVar{{Name: "A"}, {Name: "B"}})
	if a != "A;B" || b != "A;B" {
		t.Fatalf("got %q and %q, want A;B twice", a, b)
	}
	if MarkerValue(nil) != "" {
		t.Fatal("no vars must render an empty marker")
	}
}

// Retired is the read-only mirror of the sweep for --check and doctor: it
// names what Persist would DELETE — a leftover still in the store. A name the
// marker lists that a hand edit already removed is not "still persisted"
// (CodeRabbit on #1378); that is a stale record, MarkerStale's business.
func TestRetired(t *testing.T) {
	vars := []ResolvedVar{{Name: "A", Value: "1"}}
	got, err := Retired(&fakeUserEnv{values: map[string]string{ManagedMarker: "A;OLD", "OLD": "x"}}, vars)
	if err != nil || !reflect.DeepEqual(got, []string{"OLD"}) {
		t.Fatalf("got %v, %v; want [OLD]", got, err)
	}
	got, err = Retired(&fakeUserEnv{values: map[string]string{ManagedMarker: "A;OLD"}}, vars)
	if err != nil || len(got) != 0 {
		t.Fatalf("a leftover already gone from the store is not retired-and-persisted, got %v, %v", got, err)
	}
	got, err = Retired(&fakeUserEnv{values: map[string]string{}}, vars)
	if err != nil || len(got) != 0 {
		t.Fatalf("no marker must retire nothing, got %v, %v", got, err)
	}
	if _, err := Retired(&fakeUserEnv{getErr: errors.New("boom")}, vars); err == nil || !contains(err.Error(), ManagedMarker) {
		t.Fatalf("a read error must name the marker, got %v", err)
	}
}

// MarkerStale is the marker half of the --check mirror: true with no marker,
// true when the record lags the contract, false when it matches.
func TestMarkerStale(t *testing.T) {
	vars := []ResolvedVar{{Name: "B", Value: "2"}, {Name: "A", Value: "1"}}
	cases := []struct {
		name   string
		values map[string]string
		want   bool
	}{
		{"no marker", map[string]string{}, true},
		{"marker lags the contract", map[string]string{ManagedMarker: "A;OLD"}, true},
		{"marker matches", map[string]string{ManagedMarker: "A;B"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MarkerStale(&fakeUserEnv{values: tc.values}, vars)
			if err != nil || got != tc.want {
				t.Fatalf("got %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}

// A leftover a hand edit already removed: nothing is deleted, nothing is
// reported as removed, and the record is rewritten so the next --check is
// clean — the mirror stays exact in both directions.
func TestPersist_SkipsALeftoverAlreadyGone(t *testing.T) {
	store := &fakeUserEnv{values: map[string]string{"A": "1", ManagedMarker: "A;OLD"}}
	res, err := Persist([]ResolvedVar{{Name: "A", Value: "1"}}, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.deletes) != 0 {
		t.Fatalf("nothing to delete, got %v", store.deletes)
	}
	for _, r := range res {
		if r.Removed {
			t.Fatalf("an absent name must not be reported as removed: %+v", r)
		}
	}
	if store.values[ManagedMarker] != "A" {
		t.Fatalf("record must be rewritten, got %q", store.values[ManagedMarker])
	}
	stale, err := MarkerStale(store, []ResolvedVar{{Name: "A", Value: "1"}})
	if err != nil || stale {
		t.Fatalf("record must read as current after the run, got stale=%v err=%v", stale, err)
	}
}

// The marker cannot round-trip a name holding its separator: "A;B" would read
// back as A and B, and the next run could delete two unrelated values
// (CodeRabbit on #1378). Refused before anything is written.
func TestPersist_RefusesASeparatorInAName(t *testing.T) {
	store := &fakeUserEnv{values: map[string]string{"A": "theirs", "B": "theirs"}}
	_, err := Persist([]ResolvedVar{{Name: "A;B", Value: "1"}}, store)
	if err == nil || !contains(err.Error(), `"A;B"`) {
		t.Fatalf("a separator in a name must be refused naming it, got %v", err)
	}
	if len(store.ops) != 0 {
		t.Fatalf("refusal must happen before any write, got %v", store.ops)
	}
	if store.values["A"] != "theirs" || store.values["B"] != "theirs" {
		t.Fatal("foreign A and B must be untouched")
	}
}

// A delete error is surfaced with the retired name, never swallowed, and the
// results returned so far say what was done before it.
func TestPersist_DeleteErrorsNameTheVariable(t *testing.T) {
	store := &fakeUserEnv{values: map[string]string{ManagedMarker: "A;OLD", "OLD": "x"}, delErr: errors.New("access denied")}
	if _, err := Persist([]ResolvedVar{{Name: "A", Value: "1"}}, store); err == nil || !contains(err.Error(), "OLD") {
		t.Fatalf("delete error must name the retired variable, got %v", err)
	}
}

// The fake and the registry store agree on the interface's one non-obvious
// promise. If this fails, AC5 would pass locally and the second run on a real
// box would fail on a name a hand edit had already removed.
func TestFakeUserEnv_DeleteAbsentSucceeds(t *testing.T) {
	store := &fakeUserEnv{values: map[string]string{}}
	if err := store.Delete("NEVER_THERE"); err != nil {
		t.Fatalf("deleting an absent name must succeed, got %v", err)
	}
}
