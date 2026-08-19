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

// --- EnvFor: the door the Verifier seam did not close on its own ---------------
//
// Every case below exists because the first cut of this backend shipped a Verifier
// consulted by Verify and NOT by EnvFor, so `dotf secrets run` — which resolves
// every entry — hit the root's refusing resolver and failed outright. Verify was
// green, the unit tests were green, and the command was broken. Found by the
// adversarial review of OPS-026, not by this suite; these are the cases that would
// have caught it.

func TestEnvFor_SkipsTheRootWhenResolvingEverything(t *testing.T) {
	l := loaderFor(t)
	entries := []Entry{
		{Var: "ORDINARY", Backend: BackendAge, File: "ordinary"},
		{Var: "AGE_KEY_PERSONAL", Backend: BackendFileAuthority, IsFile: true,
			Dest: filepath.Join(t.TempDir(), "key.txt"), Mode: 0o600},
	}

	env, err := l.EnvFor(entries, nil)
	if err != nil {
		t.Fatalf("resolving everything must not fail because the root cannot be resolved: %v", err)
	}
	for _, kv := range env {
		if strings.HasPrefix(kv, "AGE_KEY_PERSONAL=") {
			t.Fatal("the age root must never reach a child process's environment")
		}
	}
	if len(env) != 1 || !strings.HasPrefix(env[0], "ORDINARY=") {
		t.Fatalf("the other secrets must still resolve, got %v", env)
	}
}

func TestEnvFor_RefusesTheRootWhenNamedExplicitly(t *testing.T) {
	// The skip is for a bulk request nobody aimed at the root. An explicit
	// --only AGE_KEY_PERSONAL is a question, and answering it with silence — an
	// empty environment and exit 0 — is the failure mode this repo keeps finding.
	l := loaderFor(t)
	entries := []Entry{{Var: "AGE_KEY_PERSONAL", Backend: BackendFileAuthority, IsFile: true,
		Dest: filepath.Join(t.TempDir(), "key.txt"), Mode: 0o600}}

	_, err := l.EnvFor(entries, map[string]bool{"AGE_KEY_PERSONAL": true})
	if err == nil {
		t.Fatal("naming the root explicitly must be refused out loud, not silently skipped")
	}
	if !strings.Contains(err.Error(), "age root") {
		t.Errorf("the refusal must say what it refused and why, got: %v", err)
	}
}

func TestEnvFor_ResolvesEverySingleBackendWithoutError(t *testing.T) {
	// The coverage test next door asserts a resolver EXISTS per backend. It cannot
	// see whether that resolver breaks the loop it lives in — which is exactly how
	// the blocker above passed a green suite. This asserts the loop survives one
	// entry of every declared backend.
	l := loaderFor(t)
	l.BW = stubBW{}

	var entries []Entry
	for _, b := range ValidBackends() {
		e := Entry{Var: "V_" + strings.ToUpper(strings.ReplaceAll(b, "-", "_")), Backend: b}
		switch b {
		case BackendBW:
			e.Item, e.Field = "item", "field"
		case BackendFileAuthority:
			e.IsFile, e.Dest, e.Mode = true, filepath.Join(t.TempDir(), "key.txt"), 0o600
		default:
			e.File = "src"
		}
		entries = append(entries, e)
	}

	if _, err := l.EnvFor(entries, nil); err != nil {
		t.Fatalf("EnvFor must survive one entry of every declared backend, got: %v", err)
	}
}

// stubBW is a BWReader that answers anything, so the loop-survival case above is
// about the loop and not about Bitwarden.
type stubBW struct{}

func (stubBW) Field(_, _ string) (string, error) { return "bw-value", nil }

func TestNotMaterialized_IsWhatEnvForSkipsOn(t *testing.T) {
	// EnvFor skipped on Verifier until round 2 pointed out that "answers its own
	// health question" and "must never be handed out" are different properties.
	// This pins the split: the root implements BOTH, and the skip is on the second.
	var r any = fileAuthorityResolver{}
	if _, ok := r.(NotMaterialized); !ok {
		t.Fatal("the age root must declare itself as never-materialized, which is what EnvFor skips on")
	}
	if _, ok := r.(Verifier); !ok {
		t.Fatal("the age root must also answer its own health question")
	}
	// The refusal string has one source, so the two consumers cannot drift.
	res := fileAuthorityResolver{}
	e := Entry{Var: "AGE_KEY_PERSONAL"}
	_, err := res.Resolve(e)
	if err == nil || err.Error() != res.NotMaterializedReason(e) {
		t.Errorf("Resolve must refuse with exactly the declared reason, got: %v", err)
	}
}
