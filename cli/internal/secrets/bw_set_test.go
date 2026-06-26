package secrets

import (
	"errors"
	"testing"
)

// The set write path keeps three read outcomes apart via sentinels: item absent
// (create), field absent on a present item (append), and everything else (fail). These
// tests pin the sentinels fieldFromItem/isNotFound emit.

func TestFieldFromItem_FieldNotFoundSentinel(t *testing.T) {
	const item = `{"login":{"username":"u","password":"p"},"fields":[{"name":"api-key","value":"ak"}]}`
	if _, err := fieldFromItem([]byte(item), "nonexistent"); !errors.Is(err, ErrBWFieldNotFound) {
		t.Errorf("missing custom field err = %v, want ErrBWFieldNotFound", err)
	}
	const note = `{"notes":"n","fields":[]}`
	if _, err := fieldFromItem([]byte(note), "password"); !errors.Is(err, ErrBWFieldNotFound) {
		t.Errorf("login field on a non-login item err = %v, want ErrBWFieldNotFound", err)
	}
}

func TestIsNotFound(t *testing.T) {
	for _, m := range []string{"Not found.", "not found", `bw get item "x": Not found.`} {
		if !isNotFound(m) {
			t.Errorf("isNotFound(%q) = false, want true", m)
		}
	}
	// A locked / unauthenticated vault must NOT read as absent — that is the whole point.
	for _, m := range []string{"Vault is locked.", "You are not logged in.", "mac failed."} {
		if isNotFound(m) {
			t.Errorf("isNotFound(%q) = true, want false (locked/auth is not absent)", m)
		}
	}
}

func TestNewItemBody(t *testing.T) {
	// A custom field lands by name; the item carries the requested name.
	body, err := newItemBody("openai", "api-key", "ak-1")
	if err != nil {
		t.Fatal(err)
	}
	if v, err := fieldFromItem(body, "api-key"); err != nil || v != "ak-1" {
		t.Errorf("created custom field = %q (err %v), want ak-1", v, err)
	}
	// A typed login field lands in the login block of the fresh template.
	body, err = newItemBody("svc", "password", "pw-1")
	if err != nil {
		t.Fatal(err)
	}
	if v, err := fieldFromItem(body, "password"); err != nil || v != "pw-1" {
		t.Errorf("created login field = %q (err %v), want pw-1", v, err)
	}
}
