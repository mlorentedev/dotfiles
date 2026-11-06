package env

import (
	"errors"
	"strings"
	"testing"
)

const (
	shims = `C:\Users\u\AppData\Local\mise\shims`
	bin   = `C:\Users\u\.local\bin`
)

// expandProfile stands in for registry.ExpandString: the one variable the
// cases below spell.
func expandProfile(s string) string {
	return strings.ReplaceAll(s, "%USERPROFILE%", `C:\Users\u`)
}

// MergePath owns the entries it adds, never the value: the user's entries keep
// their order and spelling, ours go to the head in the order given, and a
// result equal entry for entry to the current value is no change (W2b PR 2).
func TestMergePath(t *testing.T) {
	want := []string{shims, bin}
	cases := []struct {
		name    string
		current string
		owned   []string
		next    string
		changed bool
	}{
		{"empty Path", "", nil, shims + ";" + bin, true},
		{"already converged", shims + ";" + bin + `;C:\tools`, want, shims + ";" + bin + `;C:\tools`, false},
		{"user entries kept after ours, in order", `C:\a;C:\b`, nil, shims + ";" + bin + `;C:\a;C:\b`, true},
		{"ours in the middle move to the head", `C:\a;` + bin + `;C:\b;` + shims, nil, shims + ";" + bin + `;C:\a;C:\b`, true},
		{"wrong order is fixed", bin + ";" + shims, want, shims + ";" + bin, true},
		{"case and trailing separator match", strings.ToUpper(shims) + `\;` + bin + `\;C:\a`, want, strings.ToUpper(shims) + `\;` + bin + `\;C:\a`, false},
		{"duplicates of ours collapse", shims + ";" + bin + ";" + bin + `;C:\a`, want, shims + ";" + bin + `;C:\a`, true},
		{"a retired entry we added goes", shims + ";" + bin + `;C:\old\dotf;C:\a`, []string{shims, bin, `C:\old\dotf`}, shims + ";" + bin + `;C:\a`, true},
		{"a user entry equal to a retired one is ours to remove only when owned", `C:\old\dotf;C:\a`, nil, shims + ";" + bin + `;C:\old\dotf;C:\a`, true},
		{"empty segments alone are no change", shims + ";" + bin + `;;C:\a;`, want, shims + ";" + bin + `;;C:\a;`, false},
		{"an unexpanded spelling of ours counts as ours", `%USERPROFILE%\.local\bin;C:\a`, nil, shims + `;%USERPROFILE%\.local\bin;C:\a`, true},
		{"of two spellings of ours, the first is kept", `%USERPROFILE%\.local\bin;C:\a;` + bin, nil, shims + `;%USERPROFILE%\.local\bin;C:\a`, true},
		{"unexpanded user entries are kept verbatim", shims + ";" + bin + `;%USERPROFILE%\go\bin`, want, shims + ";" + bin + `;%USERPROFILE%\go\bin`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			next, changed := MergePath(c.current, want, c.owned, expandProfile)
			if next != c.next || changed != c.changed {
				t.Fatalf("MergePath(%q)\n got %q changed=%v\nwant %q changed=%v", c.current, next, changed, c.next, c.changed)
			}
			again, changed := MergePath(next, want, want, expandProfile)
			if changed || again != next {
				t.Fatalf("second merge is not a no-op: %q -> %q", next, again)
			}
		})
	}
}

type fakeUserPath struct {
	value  string
	writes int
	err    error
}

func (f *fakeUserPath) ReadPath() (string, error) { return f.value, f.err }
func (f *fakeUserPath) WritePath(v string) error {
	f.value = v
	f.writes++
	return nil
}

// PersistPath writes the Path and its marker once, and a second run writes
// nothing; PlanPath reports the same drift without writing.
func TestPersistPath_IdempotentWithAMarker(t *testing.T) {
	store := &fakeUserEnv{}
	path := &fakeUserPath{value: `C:\a`}
	want := []string{shims, bin}

	plan, err := PlanPath(want, store, path, expandProfile)
	if err != nil || !plan.Changed || !plan.MarkerStale || path.writes != 0 {
		t.Fatalf("plan = %+v, err %v, writes %d: want a change, a stale marker and no write", plan, err, path.writes)
	}
	if _, err := PersistPath(want, store, path, expandProfile); err != nil {
		t.Fatal(err)
	}
	if path.value != shims+";"+bin+`;C:\a` || store.values[ManagedPathMarker] != shims+";"+bin {
		t.Fatalf("Path %q, marker %q", path.value, store.values[ManagedPathMarker])
	}
	writes, sets := path.writes, len(store.writes)
	plan, err = PersistPath(want, store, path, expandProfile)
	if err != nil || plan.Changed || plan.MarkerStale || path.writes != writes || len(store.writes) != sets {
		t.Fatalf("second run wrote: plan %+v, err %v", plan, err)
	}
}

// Dropping an entry from want removes it from Path only because the marker
// says dotf added it; the marker is rewritten to what is owned now.
func TestPersistPath_RetiresOnlyWhatTheMarkerOwns(t *testing.T) {
	store := &fakeUserEnv{values: map[string]string{ManagedPathMarker: shims + ";" + bin}}
	path := &fakeUserPath{value: shims + ";" + bin + `;C:\a`}
	if _, err := PersistPath([]string{shims}, store, path, expandProfile); err != nil {
		t.Fatal(err)
	}
	if path.value != shims+`;C:\a` || store.values[ManagedPathMarker] != shims {
		t.Fatalf("Path %q, marker %q", path.value, store.values[ManagedPathMarker])
	}
}

func TestPersistPath_ReadErrorWritesNothing(t *testing.T) {
	store := &fakeUserEnv{}
	path := &fakeUserPath{err: errors.New("denied")}
	if _, err := PersistPath([]string{shims}, store, path, expandProfile); err == nil || path.writes != 0 || len(store.writes) != 0 {
		t.Fatalf("err %v, path writes %d, store writes %v", err, path.writes, store.writes)
	}
}
