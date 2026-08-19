//go:build !windows

package secrets

import (
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
