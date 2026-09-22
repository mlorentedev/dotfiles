package secrets

import (
	"encoding/json"
	"testing"
)

// A move is a read-modify-write of the whole item, like a field edit, so the
// damaging failure is the same one: a body that sets folderId and drops the
// login, the notes or a custom field. Every other key must survive untouched.
func TestSetItemFolderChangesOnlyTheFolder(t *testing.T) {
	orig := []byte(`{"id":"i1","name":"dockerhub","folderId":null,"notes":"keep",` +
		`"login":{"username":"u","password":"p"},"fields":[{"name":"PAT","value":"v","type":1}]}`)
	got, err := setItemFolder(orig, "f-apps")
	if err != nil {
		t.Fatal(err)
	}
	var a, b map[string]any
	if err := json.Unmarshal(orig, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &b); err != nil {
		t.Fatal(err)
	}
	if b["folderId"] != "f-apps" {
		t.Errorf("folderId not set: %v", b["folderId"])
	}
	delete(a, "folderId")
	delete(b, "folderId")
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Errorf("a move changed more than the folder:\n got: %s\nwant: %s", jb, ja)
	}
}

// An empty folder id unfiles the item — Bitwarden's representation is null, not
// the empty string, which the API would read as an id that does not exist.
func TestSetItemFolderEmptyUnfiles(t *testing.T) {
	got, err := setItemFolder([]byte(`{"id":"i1","folderId":"f-old"}`), "")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatal(err)
	}
	if v, ok := m["folderId"]; !ok || v != nil {
		t.Errorf("want folderId null, got %v (present %v)", v, ok)
	}
}

// Parity: the daemon writer must store exactly what the CLI writer would, for the
// same move — the same guarantee TestBWServeWriter_SetField_MatchesBWPutShape
// gives field edits. Both go through setItemFolder.
func TestBWServeWriter_MoveItem_MatchesBWPutShape(t *testing.T) {
	f, w, closeSrv := newWriterFake(t)
	defer closeSrv()
	original := append(json.RawMessage(nil), f.items["item-id-1"]...)

	if err := w.MoveItem("dockerhub", "f-apps"); err != nil {
		t.Fatalf("MoveItem: %v", err)
	}
	want, err := setItemFolder(original, "f-apps")
	if err != nil {
		t.Fatal(err)
	}
	if string(f.items["item-id-1"]) != string(want) {
		t.Fatalf("daemon move diverged from the CLI shape:\n got: %s\nwant: %s", f.items["item-id-1"], want)
	}
	if f.syncs == 0 {
		t.Error("a move must sync afterwards, or the next plan reads the item where it was")
	}
}
