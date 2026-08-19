//go:build !windows

package secrets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// Unix-only because the case needs a non-regular file that is not a directory, and
// `syscall.Mkfifo` does not exist on Windows — the whole package failed to BUILD
// there when this lived beside the portable cases. Caught by CI, not locally: a
// Linux-only verification loop cannot see a Windows compile error, which is what
// the two `test` matrix legs are for.
//
// The guard it exercises (`!fi.Mode().IsRegular()`) is portable; only the way to
// produce a subject for it is not.
func TestFileAuthority_NonRegularFileFails(t *testing.T) {
	// A FIFO at the key path carries mode 0600 and passes every other check here,
	// while `age --decrypt` fails on it. Round-2 review of OPS-026.
	dir := t.TempDir()
	p := filepath.Join(dir, "key.txt")
	if err := syscall.Mkfifo(p, 0o600); err != nil {
		t.Skipf("cannot create a FIFO here: %v", err)
	}
	err := (&Loader{}).Verify(rootEntry(t, p, 0o600))
	if err == nil {
		t.Fatal("a non-regular file at the key path must FAIL, not report OK")
	}
	if !strings.Contains(err.Error(), "regular file") {
		t.Errorf("the error must name the actual problem, got: %v", err)
	}
}

// The two mode cases live here for the same reason as the FIFO one: Windows has no
// Unix permission bits, so a 0644 key cannot exist there to be rejected. The
// platform's own behaviour is asserted in fileauthority_windows_test.go instead.

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

func TestFileAuthority_DefaultsToO600WhenNoModeDeclared(t *testing.T) {
	p := writeKey(t, 0o644)
	if err := (&Loader{}).Verify(rootEntry(t, p, 0)); err == nil {
		t.Fatal("mode 0 means 'the 0600 default', so 0644 must still FAIL")
	}
}
