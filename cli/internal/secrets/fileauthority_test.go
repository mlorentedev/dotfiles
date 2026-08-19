package secrets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The age root (#937). Every case here exists because the root's health question is
// NOT "did it resolve" — it has no source to resolve from — and a check that
// answered the easier question would report OK about a key that is world-readable,
// or absent, or a directory.

func rootEntry(t *testing.T, path string, mode os.FileMode) Entry {
	t.Helper()
	return Entry{Var: "AGE_KEY_PERSONAL", Backend: BackendFileAuthority, IsFile: true, Dest: path, Mode: mode}
}

func writeKey(t *testing.T, mode os.FileMode) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "key.txt")
	if err := os.WriteFile(p, []byte("AGE-SECRET-KEY-1TESTONLYNOTAREALKEY\n"), mode); err != nil {
		t.Fatal(err)
	}
	// WriteFile is subject to umask; set the mode explicitly or the 0644 case
	// silently becomes 0600 and the test asserts nothing.
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFileAuthority_PresentAndCorrectModeIsOK(t *testing.T) {
	p := writeKey(t, 0o600)
	if err := (&Loader{}).Verify(rootEntry(t, p, 0o600)); err != nil {
		t.Fatalf("a present 0600 key must verify, got: %v", err)
	}
}

func TestFileAuthority_WrongModeFails(t *testing.T) {
	p := writeKey(t, 0o644)
	// Confirm the mutation landed: a umask could have produced 0600 and the
	// assertion below would then pass for the wrong reason.
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o644 {
		t.Fatalf("fixture is mode %04o, not the 0644 this case is about", got)
	}

	err = (&Loader{}).Verify(rootEntry(t, p, 0o600))
	if err == nil {
		t.Fatal("a world-readable root must FAIL, not pass")
	}
	if errors.Is(err, ErrSecretAbsent) {
		t.Fatal("a wrong mode is a defect, not an absence — reporting MISSING hides it")
	}
	if !strings.Contains(err.Error(), "0644") || !strings.Contains(err.Error(), "0600") {
		t.Errorf("the error must name what it found and what it wanted, got: %v", err)
	}
}

func TestFileAuthority_AbsentIsMissingNotFailed(t *testing.T) {
	// A fresh checkout legitimately has no key yet. Same tolerance every other
	// backend gets for "not provisioned here"; calling it FAILED would make a
	// clean machine look broken and train the reader to ignore the row.
	p := filepath.Join(t.TempDir(), "nope.txt")
	err := (&Loader{}).Verify(rootEntry(t, p, 0o600))
	if !errors.Is(err, ErrSecretAbsent) {
		t.Fatalf("an absent root must be ErrSecretAbsent (MISSING), got: %v", err)
	}
}

func TestFileAuthority_EmptyFileFails(t *testing.T) {
	p := filepath.Join(t.TempDir(), "key.txt")
	if err := os.WriteFile(p, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (&Loader{}).Verify(rootEntry(t, p, 0o600)); err == nil {
		t.Fatal("a zero-byte root must FAIL — present but useless is the worst report to get right")
	}
}

func TestFileAuthority_DefaultsToO600WhenNoModeDeclared(t *testing.T) {
	p := writeKey(t, 0o644)
	if err := (&Loader{}).Verify(rootEntry(t, p, 0)); err == nil {
		t.Fatal("mode 0 means 'the 0600 default', so 0644 must still FAIL")
	}
}

func TestFileAuthority_ResolveRefuses(t *testing.T) {
	// Load-bearing. If a future change makes this resolve, `dotf secrets run`
	// starts handing the key that decrypts every other secret to child processes
	// — a widening of blast radius wearing the shape of a convenience.
	out, err := (fileAuthorityResolver{}).Resolve(rootEntry(t, "/tmp/whatever", 0o600))
	if err == nil {
		t.Fatal("the age root must never be materialized through the resolver facade")
	}
	if out != nil {
		t.Fatalf("a refusing resolver must return no bytes, got %d", len(out))
	}
}

func TestParseRegistry_FileAuthorityRejectsAnAgeSource(t *testing.T) {
	// Declaring a source names a ciphertext that cannot exist: this file is what
	// decrypts ciphertexts.
	src := `
version: 1
secrets:
  - {id: root, plane: floor, backend: file-authority, age: key, expose: {file: {var: K, path: "~/k", mode: "0600"}}}
`
	_, err := ParseRegistry([]byte(src))
	if err == nil {
		t.Fatal("file-authority with an age source must be rejected")
	}
	if !strings.Contains(err.Error(), "no age source") {
		t.Errorf("the refusal must say why, got: %v", err)
	}
}

func TestParseRegistry_FileAuthorityRejectsEnvExpose(t *testing.T) {
	// A key belongs at a path with a mode, not in an environment variable that
	// every child of every process that read it inherits.
	src := `
version: 1
secrets:
  - {id: root, plane: floor, backend: file-authority, expose: {env: AGE_KEY}}
`
	if _, err := ParseRegistry([]byte(src)); err == nil {
		t.Fatal("file-authority exposing env vars must be rejected")
	}
}
