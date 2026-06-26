package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeBW is a map-backed BWReader keyed "item/field" → value. The seam that proves
// the bw backend resolves with no Bitwarden vault and no unlock.
type fakeBW map[string]string

func (f fakeBW) Field(item, field string) (string, error) {
	if v, ok := f[item+"/"+field]; ok {
		return v, nil
	}
	return "", fmt.Errorf("fake bw: no %s/%s", item, field)
}

func TestEnvFor_BwEnvSecret(t *testing.T) {
	// Fixture values are deliberately plain words, not shaped like real provider
	// keys/tokens, so the secret scanner never flags the test data as a credential.
	l := &Loader{BW: fakeBW{"openai-item/password": "openai-test-value"}}
	entries := []Entry{{Var: "OPENAI_API_KEY", Backend: "bw", Item: "openai-item", Field: "password"}}
	env, err := l.EnvFor(entries, nil)
	if err != nil {
		t.Fatalf("EnvFor: %v", err)
	}
	if len(env) != 1 || env[0] != "OPENAI_API_KEY=openai-test-value" {
		t.Fatalf("env = %v, want [OPENAI_API_KEY=openai-test-value]", env)
	}
}

func TestEnvFor_BwFileSecret_Materialized(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "sub", "kubeconfig")
	l := &Loader{BW: fakeBW{"kube-item/notes": "apiVersion: v1\n"}}
	entries := []Entry{{Var: "KUBECONFIG", Backend: "bw", Item: "kube-item", Field: "notes", IsFile: true, Dest: dest}}
	env, err := l.EnvFor(entries, nil)
	if err != nil {
		t.Fatalf("EnvFor: %v", err)
	}
	if want := "KUBECONFIG=" + dest; len(env) != 1 || env[0] != want {
		t.Fatalf("env = %v, want [%q]", env, want)
	}
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "apiVersion: v1\n" {
		t.Errorf("materialized = %q (err %v), want content verbatim", data, err)
	}
}

// TestEnvFor_MixedBackends_OnlyFilter proves one EnvFor call dispatches age and bw
// entries to their own resolvers, and --only scopes across both.
func TestEnvFor_MixedBackends_OnlyFilter(t *testing.T) {
	l := &Loader{
		SecretsDir: t.TempDir(),
		Decrypt:    func(_, _ string) ([]byte, error) { return []byte("age-val\n"), nil },
		BW:         fakeBW{"it/password": "bw-val"},
	}
	entries := []Entry{
		{Var: "AGE_VAR", Backend: "age", File: "a"},
		{Var: "BW_VAR", Backend: "bw", Item: "it", Field: "password"},
	}
	env, err := l.EnvFor(entries, map[string]bool{"BW_VAR": true})
	if err != nil {
		t.Fatalf("EnvFor: %v", err)
	}
	if len(env) != 1 || env[0] != "BW_VAR=bw-val" {
		t.Fatalf("--only BW_VAR = %v, want [BW_VAR=bw-val]", env)
	}
}

func TestEnvFor_BwReaderError_FailsFast(t *testing.T) {
	l := &Loader{BW: fakeBW{}} // empty → Field errors
	if _, err := l.EnvFor([]Entry{{Var: "X", Backend: "bw", Item: "missing", Field: "password"}}, nil); err == nil {
		t.Fatal("expected EnvFor to fail fast when the bw reader errors")
	}
}

func TestEnvFor_BwNilReader_ClearError(t *testing.T) {
	l := &Loader{} // BW nil — Bitwarden locked / not wired
	_, err := l.EnvFor([]Entry{{Var: "X", Backend: "bw", Item: "it", Field: "password"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "bw backend unavailable") {
		t.Fatalf("nil reader err = %v, want a clear 'bw backend unavailable' message", err)
	}
}

func TestEnvFor_UnknownBackend_Errors(t *testing.T) {
	l := &Loader{}
	if _, err := l.EnvFor([]Entry{{Var: "X", Backend: "vault", File: "x"}}, nil); err == nil {
		t.Fatal("expected an error for an unknown backend")
	}
}

func TestFieldFromItem(t *testing.T) {
	const itemJSON = `{
		"login": {"username": "alice", "password": "login-test-value"},
		"notes": "multi\nline",
		"fields": [{"name": "api-key", "value": "ak-1"}, {"name": "client-id", "value": "cid-2"}]
	}`
	for _, c := range []struct{ field, want string }{
		{"password", "login-test-value"},
		{"username", "alice"},
		{"notes", "multi\nline"},
		{"api-key", "ak-1"},
		{"client-id", "cid-2"},
	} {
		got, err := fieldFromItem([]byte(itemJSON), c.field)
		if err != nil || got != c.want {
			t.Errorf("field %q = %q (err %v), want %q", c.field, got, err, c.want)
		}
	}
	if _, err := fieldFromItem([]byte(itemJSON), "nonexistent"); err == nil {
		t.Error("unknown custom field must error (catches a registry typo)")
	}
	if _, err := fieldFromItem([]byte("not json"), "password"); err == nil {
		t.Error("invalid JSON must error")
	}
}

// A typed login field requested against an item with no login block (e.g. a secure
// note) must error, not return a silent empty value (#612 A1).
func TestFieldFromItem_NonLoginItem_Errors(t *testing.T) {
	const noteJSON = `{"notes":"some note","fields":[{"name":"api-key","value":"ak-1"}]}`
	if _, err := fieldFromItem([]byte(noteJSON), "password"); err == nil {
		t.Error("field: password against a non-login item must error")
	}
	if v, err := fieldFromItem([]byte(noteJSON), "notes"); err != nil || v != "some note" {
		t.Errorf("notes = %q (err %v)", v, err)
	}
	if v, err := fieldFromItem([]byte(noteJSON), "api-key"); err != nil || v != "ak-1" {
		t.Errorf("custom field = %q (err %v)", v, err)
	}
}

// A bw field that resolves to an empty value must fail fast in EnvFor — never inject
// VAR= into the child (#612 A1).
func TestEnvFor_BwEmptyValue_FailsFast(t *testing.T) {
	l := &Loader{BW: fakeBW{"it/password": ""}}
	_, err := l.EnvFor([]Entry{{Var: "X", Backend: "bw", Item: "it", Field: "password"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "empty value") {
		t.Fatalf("empty bw value must fail fast, got err=%v", err)
	}
}
