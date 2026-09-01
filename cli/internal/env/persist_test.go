package env

import (
	"errors"
	"testing"
)

// fakeUserEnv records what Persist wrote, so the tests assert the idempotence
// contract (touch only what differs) without a registry.
type fakeUserEnv struct {
	values  map[string]string
	writes  []string
	deletes []string
	// ops records every write in order ("set:NAME", "del:NAME") so a test can
	// pin the sweep-before-write order Persist depends on.
	ops    []string
	getErr error
	setErr error
	delErr error
}

func (f *fakeUserEnv) Get(name string) (string, bool, error) {
	if f.getErr != nil {
		return "", false, f.getErr
	}
	v, ok := f.values[name]
	return v, ok, nil
}

func (f *fakeUserEnv) Set(name, value string) error {
	if f.setErr != nil {
		return f.setErr
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[name] = value
	f.writes = append(f.writes, name)
	f.ops = append(f.ops, "set:"+name)
	return nil
}

// Delete honours the interface's promise: an absent name is success. The
// registry store maps ErrNotExist to nil for the same reason, and
// TestFakeUserEnv_DeleteAbsentSucceeds keeps the fake honest about it.
func (f *fakeUserEnv) Delete(name string) error {
	if f.delErr != nil {
		return f.delErr
	}
	delete(f.values, name)
	f.deletes = append(f.deletes, name)
	f.ops = append(f.ops, "del:"+name)
	return nil
}

// Persist writes only what differs: missing and different values are written,
// equal ones are left alone, and a second run writes nothing (CLI-058, #1324).
func TestPersist_TouchesOnlyWhatDiffers(t *testing.T) {
	vars := []ResolvedVar{
		{Name: "DOTFILES_REPO_DIR", Value: `C:\Users\u\Projects\dotfiles`},
		{Name: "DOTFILES_DIR", Value: `C:\Users\u\.dotfiles`},
		{Name: "VAULT_PATH", Value: `C:\Users\u\Projects\knowledge`},
	}
	store := &fakeUserEnv{values: map[string]string{
		"DOTFILES_DIR": `C:\Users\u\.dotfiles`, // equal → untouched
		"VAULT_PATH":   `C:\Users\u\old-vault`, // different → rewritten
	}}

	res, err := Persist(vars, store)
	if err != nil {
		t.Fatal(err)
	}
	changed := map[string]bool{}
	for _, r := range res {
		changed[r.Name] = r.Changed
	}
	if !changed["DOTFILES_REPO_DIR"] || changed["DOTFILES_DIR"] || !changed["VAULT_PATH"] {
		t.Fatalf("changed flags wrong: %+v", changed)
	}
	// The two differing vars, then the marker recording all three (CLI-065).
	if len(store.writes) != 3 || store.writes[2] != ManagedMarker {
		t.Fatalf("expected exactly the two differing vars and then the marker to be written, got %v", store.writes)
	}

	// Second run: nothing differs, nothing is written.
	store.writes = nil
	res, err = Persist(vars, store)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res {
		if r.Changed {
			t.Fatalf("second run must change nothing, got %+v", r)
		}
	}
	if len(store.writes) != 0 {
		t.Fatalf("second run wrote %v", store.writes)
	}
}

// Drift is the read-only mirror of Persist: it names what Persist would write.
func TestDrift_NamesMissingAndDifferent(t *testing.T) {
	vars := []ResolvedVar{
		{Name: "A", Value: "1"}, {Name: "B", Value: "2"}, {Name: "C", Value: "3"},
	}
	store := &fakeUserEnv{values: map[string]string{"A": "1", "B": "x"}}
	d, err := Drift(vars, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 2 || d[0].Name != "B" || d[1].Name != "C" {
		t.Fatalf("drift must list B (different) and C (missing), got %+v", d)
	}
}

// A store error is surfaced with the variable's name, never swallowed.
func TestPersist_StoreErrorsNameTheVariable(t *testing.T) {
	vars := []ResolvedVar{{Name: "DOTFILES_DIR", Value: "x"}}
	if _, err := Persist(vars, &fakeUserEnv{values: map[string]string{}, setErr: errors.New("access denied")}); err == nil || !contains(err.Error(), "DOTFILES_DIR") {
		t.Fatalf("set error must name the variable, got %v", err)
	}
	if _, err := Drift(vars, &fakeUserEnv{getErr: errors.New("boom")}); err == nil || !contains(err.Error(), "DOTFILES_DIR") {
		t.Fatalf("get error must name the variable, got %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
