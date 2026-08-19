package secrets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The root-drift check (#1000 AC3). It answers the one question the other checks
// cannot: is the key on this disk still the key that was declared? A root replaced,
// truncated, or restored from the wrong backup passes present-regular-0600 without
// complaint, and stays indistinguishable from a healthy one until the day it is
// needed.

const (
	declaredRecipient = "age1declaredrecipientnotarealkeyjustafixture000000000000000"
	otherRecipient    = "age1somethingelseentirelyalsonotarealkey00000000000000000000"
)

func rootWithRecipient(t *testing.T, recipient string) Entry {
	t.Helper()
	p := filepath.Join(t.TempDir(), "key.txt")
	if err := os.WriteFile(p, []byte("AGE-SECRET-KEY-1TESTONLYNOTAREALKEY\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	return Entry{Var: "AGE_KEY_PERSONAL", Backend: BackendFileAuthority, IsFile: true,
		Dest: p, Mode: 0o600, Recipient: recipient}
}

func loaderDeriving(r string, err error) *Loader {
	return &Loader{Recipient: func(string) (string, error) { return r, err }}
}

func TestDrift_MatchingRecipientVerifies(t *testing.T) {
	e := rootWithRecipient(t, declaredRecipient)
	if err := loaderDeriving(declaredRecipient, nil).Verify(e); err != nil {
		t.Fatalf("a key deriving the declared recipient must verify, got: %v", err)
	}
}

func TestDrift_MismatchFailsAndNamesBoth(t *testing.T) {
	e := rootWithRecipient(t, declaredRecipient)
	err := loaderDeriving(otherRecipient, nil).Verify(e)
	if err == nil {
		t.Fatal("a key that is not the declared one must FAIL")
	}
	if errors.Is(err, ErrSecretAbsent) {
		t.Fatal("a drifted key is a defect, not an absence — MISSING would hide it")
	}
	// Both halves, because "they differ" without saying how sends the reader to the
	// wrong place: the fix depends on WHICH one is wrong.
	for _, want := range []string{declaredRecipient, otherRecipient} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error must name %q, got: %v", want, err)
		}
	}
}

func TestDrift_UndeclaredMeansNotChecked(t *testing.T) {
	// Deliberate no-op, not a silent pass: with no `recipient:` there is nothing to
	// compare against, and inventing a comparison would be worse than not making one.
	e := rootWithRecipient(t, "")
	called := false
	l := &Loader{Recipient: func(string) (string, error) { called = true; return otherRecipient, nil }}
	if err := l.Verify(e); err != nil {
		t.Fatalf("an undeclared recipient must verify exactly as before, got: %v", err)
	}
	if called {
		t.Error("nothing should be derived when nothing was declared")
	}
}

func TestDrift_CannotDeriveIsReportedNotSwallowed(t *testing.T) {
	// age-keygen missing, unreadable key, anything: a check that cannot run must say
	// so. Returning nil here would turn "the tool is absent" into "the key is fine",
	// which is this check's own failure arriving through the check itself.
	e := rootWithRecipient(t, declaredRecipient)
	err := loaderDeriving("", errors.New("age-keygen: executable file not found in $PATH")).Verify(e)
	if err == nil {
		t.Fatal("a check that could not run must not report OK")
	}
	if !strings.Contains(err.Error(), "age-keygen") {
		t.Errorf("the failure must carry the underlying cause, got: %v", err)
	}
}

func TestParseRegistry_RecipientOnlyOnFileAuthority(t *testing.T) {
	// Accepting it elsewhere would be the worse failure: a reader would take the key
	// as pinned while nothing compares it — a declaration that lies.
	src := `
version: 1
secrets:
  - {id: s, plane: app, backend: age, age: src, recipient: age1abc, expose: {env: V}}
`
	err := parseErr(t, src)
	if !strings.Contains(err.Error(), "only meaningful on a file-authority") {
		t.Errorf("the refusal must say why, got: %v", err)
	}
}

func TestParseRegistry_RecipientMustLookLikeARecipient(t *testing.T) {
	// A private key pasted here by accident is the mistake worth catching: it would
	// commit the secret this whole file exists to protect.
	const pastedByMistake = "AGE-SECRET-KEY-1THISWOULDBETHEACTUALPRIVATEKEY"
	src := `
version: 1
secrets:
  - {id: s, plane: floor, backend: file-authority, recipient: ` + pastedByMistake + `, expose: {file: {var: K, path: "~/k", mode: "0600"}}}
`
	err := parseErr(t, src)
	if !strings.Contains(err.Error(), "age public recipient") {
		t.Errorf("the refusal must name what was expected, got: %v", err)
	}
	// And it must NOT echo the value. The likeliest way to reach this branch is
	// pasting the private key, and an error message reaches terminal scrollback and
	// CI logs — printing it would commit the one secret this field protects.
	if strings.Contains(err.Error(), pastedByMistake) {
		t.Fatalf("the refusal echoed the rejected value, which may be a private key: %v", err)
	}
}

func parseErr(t *testing.T, src string) error {
	t.Helper()
	_, err := ParseRegistry([]byte(src))
	if err == nil {
		t.Fatal("expected the parser to refuse this registry")
	}
	return err
}
