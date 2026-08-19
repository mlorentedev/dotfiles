//go:build windows

package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

// The declared mode is not applicable on Windows, and this asserts that we say so
// by BEHAVIOUR rather than only in a comment: a key whose Unix bits would read as
// world-readable still verifies, because those bits are a synthesised number no
// caller set and rejecting them would fail every healthy Windows machine.
//
// The cases that DO carry meaning on both platforms — present, regular, non-empty —
// are in the portable file and run here too.
func TestFileAuthority_ModeIsNotApplicableOnWindows(t *testing.T) {
	p := filepath.Join(t.TempDir(), "key.txt")
	if err := os.WriteFile(p, []byte("AGE-SECRET-KEY-1TESTONLYNOTAREALKEY\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := Entry{Var: "AGE_KEY_PERSONAL", Backend: BackendFileAuthority, IsFile: true, Dest: p, Mode: 0o600}
	if err := (&Loader{}).Verify(e); err != nil {
		t.Fatalf("Windows has no Unix permission bits, so the declared mode must not fail a healthy key: %v", err)
	}
}

// An absent key is still MISSING and an empty one still FAILS here — the checks
// that are portable must not have been lost along with the one that is not.
func TestFileAuthority_PortableChecksStillRunOnWindows(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	e := Entry{Var: "K", Backend: BackendFileAuthority, IsFile: true, Dest: empty, Mode: 0o600}
	if err := (&Loader{}).Verify(e); err == nil {
		t.Fatal("a zero-byte root must still FAIL on Windows")
	}
}
