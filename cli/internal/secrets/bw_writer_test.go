package secrets

import (
	"encoding/json"
	"testing"
)

const writerItem = `{
  "id": "abc-123",
  "type": 1,
  "name": "x-twitter",
  "notes": "old note",
  "login": { "username": "example-user", "password": "example-old" },
  "fields": [
    { "name": "api-key", "value": "old-ak", "type": 1 },
    { "name": "client-id", "value": "cid", "type": 1 }
  ]
}`

// readBack reads a field from a setItemField result via the read path, proving the
// write round-trips; preserved checks the untouched keys/fields.
func assertPreserved(t *testing.T, out []byte) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("output not valid JSON: %v", err)
	}
	if m["id"] != "abc-123" || m["name"] != "x-twitter" || m["type"] != float64(1) {
		t.Errorf("top-level keys not preserved: id=%v name=%v type=%v", m["id"], m["name"], m["type"])
	}
}

func TestSetItemField_TypedLogin_PreservesSiblings(t *testing.T) {
	out, err := setItemField([]byte(writerItem), "password", "example-new")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := fieldFromItem(out, "password"); v != "example-new" {
		t.Errorf("password = %q, want example-new", v)
	}
	for _, c := range []struct{ field, want string }{
		{"username", "example-user"}, {"notes", "old note"}, {"api-key", "old-ak"}, {"client-id", "cid"},
	} {
		if v, _ := fieldFromItem(out, c.field); v != c.want {
			t.Errorf("sibling %s clobbered = %q, want %q", c.field, v, c.want)
		}
	}
	assertPreserved(t, out)
}

func TestSetItemField_CreatesLoginIfAbsent(t *testing.T) {
	const note = `{"id":"n1","type":2,"name":"secure-note","notes":"hi"}`
	out, err := setItemField([]byte(note), "password", "example-pass")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := fieldFromItem(out, "password"); v != "example-pass" {
		t.Errorf("password = %q, want example-pass (login created)", v)
	}
	if v, _ := fieldFromItem(out, "notes"); v != "hi" {
		t.Errorf("notes clobbered = %q", v)
	}
}

func TestSetItemField_Notes(t *testing.T) {
	out, err := setItemField([]byte(writerItem), "notes", "new note")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := fieldFromItem(out, "notes"); v != "new note" {
		t.Errorf("notes = %q, want new note", v)
	}
	if v, _ := fieldFromItem(out, "password"); v != "example-old" {
		t.Errorf("login clobbered by a notes write = %q", v)
	}
}

func TestSetItemField_CustomFieldUpdate(t *testing.T) {
	out, err := setItemField([]byte(writerItem), "api-key", "new-ak")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := fieldFromItem(out, "api-key"); v != "new-ak" {
		t.Errorf("api-key = %q, want new-ak", v)
	}
	if v, _ := fieldFromItem(out, "client-id"); v != "cid" {
		t.Errorf("sibling custom field clobbered = %q", v)
	}
}

func TestSetItemField_CustomFieldAppend(t *testing.T) {
	out, err := setItemField([]byte(writerItem), "bearer-token", "bt-1")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := fieldFromItem(out, "bearer-token"); v != "bt-1" {
		t.Errorf("appended field bearer-token = %q, want bt-1", v)
	}
	// the existing custom fields survive the append
	if v, _ := fieldFromItem(out, "api-key"); v != "old-ak" {
		t.Errorf("existing field clobbered by append = %q", v)
	}
}

func TestSetItemField_BadJSON(t *testing.T) {
	if _, err := setItemField([]byte("not json"), "password", "x"); err == nil {
		t.Error("malformed item JSON must error")
	}
}

func TestItemID(t *testing.T) {
	if id, err := itemID([]byte(writerItem)); err != nil || id != "abc-123" {
		t.Errorf("itemID = %q (err %v), want abc-123", id, err)
	}
	if _, err := itemID([]byte(`{"name":"no-id"}`)); err == nil {
		t.Error("an item with no id must error")
	}
}

// fakeBWWriter records writes so command tests can assert without a real vault.
type fakeBWWriter map[string]string // "item/field" -> value

func (f fakeBWWriter) SetField(item, field, value string) error {
	f[item+"/"+field] = value
	return nil
}

func TestBWWriter_MockRoundTrip(t *testing.T) {
	var w BWWriter = fakeBWWriter{}
	if err := w.SetField("it", "password", "v"); err != nil {
		t.Fatal(err)
	}
	if w.(fakeBWWriter)["it/password"] != "v" {
		t.Error("fake BWWriter did not record the write")
	}
}
