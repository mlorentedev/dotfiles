package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readMachineDoc(t *testing.T, path string) map[string]json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read machine.json: %v", err)
	}
	doc := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse machine.json: %v", err)
	}
	return doc
}

// A machine with no file at all acquires an identity without a hand edit
// (#1236 AC1), and the policy reader then sees it.
func TestSetMachineIdentity_DeclaresOnAFreshMachine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dotfiles", "machine.json")

	changed, err := SetMachineIdentity(path, "macmini", false)
	if err != nil || !changed {
		t.Fatalf("SetMachineIdentity = %v, %v; want true, nil", changed, err)
	}
	p, err := LoadMachinePolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Identified || p.ID != "macmini" {
		t.Fatalf("policy = %+v, want identified as macmini", p)
	}
}

// Re-running it on a declared machine is changed=0 and never rewrites the
// file (#1236 AC2).
func TestSetMachineIdentity_SameIDIsIdempotent(t *testing.T) {
	path := seedMachineFile(t, `{"machine": {"id": "msi"}}`)
	before, _ := os.ReadFile(path)

	changed, err := SetMachineIdentity(path, "msi", false)
	if err != nil || changed {
		t.Fatalf("SetMachineIdentity = %v, %v; want false, nil", changed, err)
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Errorf("an unchanged identity rewrote the file:\n%s\nwas:\n%s", after, before)
	}
}

// A different id is a rename, not a re-run: it is refused unless asked for, so
// provisioning can never overwrite what an owner declared (#1236 AC2).
func TestSetMachineIdentity_DifferentIDNeedsReplace(t *testing.T) {
	path := seedMachineFile(t, `{"machine": {"id": "msi"}}`)

	changed, err := SetMachineIdentity(path, "other", false)
	if err == nil || changed {
		t.Fatalf("SetMachineIdentity = %v, %v; want a refusal", changed, err)
	}
	for _, want := range []string{`"msi"`, "--replace"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q lacks %q", err, want)
		}
	}
	if p, _ := LoadMachinePolicy(path); p.ID != "msi" {
		t.Errorf("a refused rename changed the id to %q", p.ID)
	}

	changed, err = SetMachineIdentity(path, "other", true)
	if err != nil || !changed {
		t.Fatalf("with replace: %v, %v; want true, nil", changed, err)
	}
	if p, _ := LoadMachinePolicy(path); p.ID != "other" {
		t.Errorf("id = %q after --replace, want other", p.ID)
	}
}

// machine.json holds sections other readers own (lesson 402): declaring an id
// must keep paths, roles, pools and every other key of the machine block.
func TestSetMachineIdentity_KeepsEveryOtherSection(t *testing.T) {
	path := seedMachineFile(t, `{
  "paths": {"DOTFILES_REPO_DIR": "/r"},
  "roles": ["ssh-target"],
  "pools": {"deny": ["nan"]},
  "machine": {"note": "kept"}
}`)

	if _, err := SetMachineIdentity(path, "msi", false); err != nil {
		t.Fatal(err)
	}
	doc := readMachineDoc(t, path)
	for _, k := range []string{"paths", "roles", "pools"} {
		if _, ok := doc[k]; !ok {
			t.Errorf("section %q was dropped: %v", k, doc)
		}
	}
	var machine map[string]string
	if err := json.Unmarshal(doc["machine"], &machine); err != nil {
		t.Fatal(err)
	}
	if machine["id"] != "msi" || machine["note"] != "kept" {
		t.Errorf("machine block = %v, want id msi and note kept", machine)
	}
	if p, _ := LoadMachinePolicy(path); len(p.Deny) != 1 || p.Deny[0] != "nan" {
		t.Errorf("deny list = %v, want [nan]", p.Deny)
	}
}

// An id nothing could read back, or one that reads as a different value, is
// refused before it reaches the file.
func TestSetMachineIdentity_RejectsAnIDThatIsNotALabel(t *testing.T) {
	for _, id := range []string{"", "   ", "two words", "tab\there", "new\nline", "quo\"te", strings.Repeat("a", 65)} {
		path := filepath.Join(t.TempDir(), "machine.json")
		if changed, err := SetMachineIdentity(path, id, false); err == nil || changed {
			t.Errorf("id %q = %v, %v; want a refusal", id, changed, err)
		}
		if _, err := os.Stat(path); err == nil {
			t.Errorf("id %q created machine.json", id)
		}
	}
}

// A file that cannot be parsed is an error, never an identity written over it.
func TestSetMachineIdentity_MalformedFileIsAnError(t *testing.T) {
	path := seedMachineFile(t, `{not json`)
	if changed, err := SetMachineIdentity(path, "msi", false); err == nil || changed {
		t.Fatalf("SetMachineIdentity = %v, %v; want a parse error", changed, err)
	}
}
