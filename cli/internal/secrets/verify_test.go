package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_Verify_Classification(t *testing.T) {
	cases := []struct {
		name       string
		dec        Decryptor
		wantErr    bool
		wantAbsent bool
	}{
		{"ok", func(string, string) ([]byte, error) { return []byte("value\n"), nil }, false, false},
		{"absent", func(string, string) ([]byte, error) { return nil, fmt.Errorf("%w: f", ErrSecretAbsent) }, true, true},
		{"empty", func(string, string) ([]byte, error) { return []byte("\n"), nil }, true, false},
		{"error", func(string, string) ([]byte, error) { return nil, fmt.Errorf("vault locked") }, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := &Loader{SecretsDir: t.TempDir(), Decrypt: c.dec}
			err := l.Verify(Entry{Var: "X", Backend: "age", File: "x"})
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, c.wantErr)
			}
			if c.wantAbsent && !errors.Is(err, ErrSecretAbsent) {
				t.Errorf("expected ErrSecretAbsent, got %v", err)
			}
			if !c.wantAbsent && errors.Is(err, ErrSecretAbsent) {
				t.Errorf("did not expect ErrSecretAbsent, got %v", err)
			}
		})
	}
}

// Verify must resolve a file secret WITHOUT writing it to disk (no side effect).
func TestLoader_Verify_FileSecret_NotMaterialized(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "sub", "kubeconfig")
	l := &Loader{
		SecretsDir: t.TempDir(),
		Decrypt:    func(string, string) ([]byte, error) { return []byte("content\n"), nil },
	}
	if err := l.Verify(Entry{Var: "KUBECONFIG", Backend: "age", File: "k", IsFile: true, Dest: dest}); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if _, err := os.Stat(dest); err == nil {
		t.Error("Verify must NOT materialize the file secret to disk")
	}
}

func TestLoader_Verify_Bw(t *testing.T) {
	if err := (&Loader{BW: fakeBW{"it/password": "v"}}).Verify(
		Entry{Var: "X", Backend: "bw", Item: "it", Field: "password"}); err != nil {
		t.Errorf("bw ok Verify: %v", err)
	}
	if err := (&Loader{BW: fakeBW{"it/password": ""}}).Verify(
		Entry{Var: "X", Backend: "bw", Item: "it", Field: "password"}); err == nil {
		t.Error("an empty bw value must fail Verify")
	}
	if err := (&Loader{BW: fakeBW{}}).Verify(
		Entry{Var: "X", Backend: "bw", Item: "nope", Field: "password"}); err == nil {
		t.Error("a missing bw item must fail Verify")
	}
}
