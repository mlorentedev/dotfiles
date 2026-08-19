//go:build !windows

package secrets

import (
	"fmt"
	"io/fs"
	"os"
)

// checkKeyMode enforces the declared permission bits on the age root.
//
// Unix only, and that is a statement about the filesystem rather than about
// effort. Windows has no Unix permission bits: os.Chmod there toggles the
// read-only attribute and nothing else, and fi.Mode().Perm() reports a
// synthesised 0666 or 0444 that no caller set. Enforcing 0600 against that
// number would fail every healthy Windows machine — a gate that is wrong about
// a whole class of machine teaches its readers to discount it, which is the
// failure GUARD-002 exists to prevent, arrived at from the other side.
//
// The Windows half returns nil and says the same thing from the other file, so
// the asymmetry is declared in both places rather than inferred from one.
func checkKeyMode(fi os.FileInfo, path string, want fs.FileMode) error {
	if want == 0 {
		want = 0o600
	}
	if got := fi.Mode().Perm(); got != want.Perm() {
		return fmt.Errorf("%s has mode %04o, expected %04o", path, got, want.Perm())
	}
	return nil
}
