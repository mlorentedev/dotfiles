package secrets

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

// newWriterFake returns a fake daemon holding one login item, plus the writer under
// test wired to it.
func newWriterFake(t *testing.T) (*fakeBWServe, BWServeWriter, func()) {
	t.Helper()
	f := &fakeBWServe{
		status: "unlocked",
		names:  map[string]string{"item-id-1": "dockerhub"},
		items: map[string]json.RawMessage{
			"item-id-1": json.RawMessage(`{"id":"item-id-1","type":1,"name":"dockerhub",` +
				`"notes":"keep me","login":{"username":"manu","password":"old-secret"},` +
				`"fields":[{"name":"PAT","value":"old-pat","type":1}]}`),
		},
	}
	srv := httptest.NewServer(f.handler())
	return f, BWServeWriter{Client: BWServeClient{BaseURL: srv.URL}}, srv.Close
}

// TestBWServeWriter_SetField_UpdatesAndPreserves: a write must change exactly one field
// and leave every other key intact. It is read-modify-write, so a sloppy implementation
// that PUTs a minimal body silently destroys the item's other fields — the most
// damaging failure available on this path, and one that only shows up later.
func TestBWServeWriter_SetField_UpdatesAndPreserves(t *testing.T) {
	f, w, closeSrv := newWriterFake(t)
	defer closeSrv()

	if err := w.SetField("dockerhub", "password", "new-secret"); err != nil {
		t.Fatalf("SetField: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(f.items["item-id-1"], &got); err != nil {
		t.Fatalf("stored item is not valid JSON: %v", err)
	}
	login, _ := got["login"].(map[string]any)
	if login["password"] != "new-secret" {
		t.Fatalf("password = %v, want new-secret", login["password"])
	}
	if login["username"] != "manu" {
		t.Fatalf("sibling username was clobbered: %v", login["username"])
	}
	if got["notes"] != "keep me" {
		t.Fatalf("notes were clobbered: %v", got["notes"])
	}
	if got["id"] != "item-id-1" || got["name"] != "dockerhub" {
		t.Fatalf("identity keys were clobbered: id=%v name=%v", got["id"], got["name"])
	}
	fields, _ := got["fields"].([]any)
	if len(fields) != 1 {
		t.Fatalf("custom fields were clobbered: %v", fields)
	}
}

// TestBWServeWriter_SetField_MatchesBWPutShape is the parity assertion that keeps the
// two backends interchangeable: for the same item and the same edit, the daemon writer
// must store byte-identical JSON to what the CLI writer would have produced. Both go
// through setItemField, and this test is what stops a future change from "optimising"
// one of them into divergence.
func TestBWServeWriter_SetField_MatchesBWPutShape(t *testing.T) {
	f, w, closeSrv := newWriterFake(t)
	defer closeSrv()
	original := append(json.RawMessage(nil), f.items["item-id-1"]...)

	if err := w.SetField("dockerhub", "PAT", "new-pat"); err != nil {
		t.Fatalf("SetField: %v", err)
	}

	want, err := setItemField(original, "PAT", "new-pat")
	if err != nil {
		t.Fatalf("reference setItemField: %v", err)
	}
	if string(f.items["item-id-1"]) != string(want) {
		t.Fatalf("daemon write diverged from the CLI shape:\n got: %s\nwant: %s", f.items["item-id-1"], want)
	}
}

// TestBWServeWriter_SetField_AbsentItemIsClassified: `set` decides whether to offer
// creation by testing for ErrBWItemNotFound. If the daemon writer reported absence as a
// generic error, `set` would refuse to create against a daemon while creating happily
// against the CLI — the same asymmetry BUG-084 is about, one level up. The security
// crux of #612 also depends on this: a LOCKED vault must never look like an absent item.
func TestBWServeWriter_SetField_AbsentItemIsClassified(t *testing.T) {
	_, w, closeSrv := newWriterFake(t)
	defer closeSrv()

	err := w.SetField("no-such-item", "password", "v")
	if !errors.Is(err, ErrBWItemNotFound) {
		t.Fatalf("absent item must be ErrBWItemNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "dotf secrets set") {
		t.Fatalf("absence should name the remediation, got: %v", err)
	}
}

// TestBWServeWriter_CreateItem_SendsRawJSON: the daemon takes a raw JSON body where the
// CLI needs base64-on-stdin. A regression to base64 would send a JSON string where an
// object is expected; the fake rejects anything that is not valid JSON, so this pins the
// one genuine behavioural difference between the two writers.
func TestBWServeWriter_CreateItem_SendsRawJSON(t *testing.T) {
	f, w, closeSrv := newWriterFake(t)
	defer closeSrv()

	if err := w.CreateItem("brand-new", "password", "v1", "folder-id-9"); err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if len(f.created) != 1 {
		t.Fatalf("expected exactly one POST /object/item, got %d", len(f.created))
	}

	var got map[string]any
	if err := json.Unmarshal(f.created[0], &got); err != nil {
		t.Fatalf("created body is not raw JSON (base64 regression?): %v", err)
	}
	if got["name"] != "brand-new" {
		t.Fatalf("name = %v", got["name"])
	}
	if got["folderId"] != "folder-id-9" {
		t.Fatalf("folderId = %v, want folder-id-9 (OPS-028 taxonomy)", got["folderId"])
	}
	login, _ := got["login"].(map[string]any)
	if login["password"] != "v1" {
		t.Fatalf("password = %v", login["password"])
	}

	want, err := newItemBody("brand-new", "password", "v1", "folder-id-9")
	if err != nil {
		t.Fatalf("reference newItemBody: %v", err)
	}
	if string(f.created[0]) != string(want) {
		t.Fatalf("created item diverged from the CLI shape:\n got: %s\nwant: %s", f.created[0], want)
	}
}

// TestBWServeWriter_ResolveFolder covers the taxonomy half: an existing folder resolves
// to its id without creating anything, an absent one is created, and an empty name is a
// no-op rather than an error (so callers never special-case the unfoldered secret).
func TestBWServeWriter_ResolveFolder(t *testing.T) {
	t.Run("existing folder resolves without creating", func(t *testing.T) {
		f, w, closeSrv := newWriterFake(t)
		defer closeSrv()
		f.folders = map[string]string{"fid-apps": "apps", "fid-infra": "infra"}

		id, err := w.ResolveFolder("apps")
		if err != nil {
			t.Fatalf("ResolveFolder: %v", err)
		}
		if id != "fid-apps" {
			t.Fatalf("id = %q, want fid-apps", id)
		}
		if len(f.folders) != 2 {
			t.Fatalf("resolving an existing folder must not create one: %v", f.folders)
		}
	})

	t.Run("absent folder is created", func(t *testing.T) {
		f, w, closeSrv := newWriterFake(t)
		defer closeSrv()
		f.folders = map[string]string{"fid-apps": "apps"}
		f.nextID = "fid-new"

		id, err := w.ResolveFolder("infra")
		if err != nil {
			t.Fatalf("ResolveFolder: %v", err)
		}
		if id != "fid-new" {
			t.Fatalf("id = %q, want fid-new", id)
		}
		if f.folders["fid-new"] != "infra" {
			t.Fatalf("folder was not created: %v", f.folders)
		}
	})

	t.Run("empty name is a no-op", func(t *testing.T) {
		f, w, closeSrv := newWriterFake(t)
		defer closeSrv()

		id, err := w.ResolveFolder("")
		if err != nil {
			t.Fatalf("empty name must not error: %v", err)
		}
		if id != "" {
			t.Fatalf("id = %q, want empty", id)
		}
		if len(f.folders) != 0 {
			t.Fatalf("empty name must not create a folder: %v", f.folders)
		}
	})
}

// TestBWServeWriter_SatisfiesWriteClient is a compile-time guarantee that the daemon
// writer covers the same surface as BWPut. Without it, a half-implemented backend fails
// only at the call site that happens to use the missing method.
func TestBWServeWriter_SatisfiesWriteClient(t *testing.T) {
	var _ BWWriteClient = BWServeWriter{}
	var _ BWWriteClient = BWPut{}
}
