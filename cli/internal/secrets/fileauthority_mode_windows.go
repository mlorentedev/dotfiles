//go:build windows

package secrets

import (
	"io/fs"
	"os"
)

// checkKeyMode is a no-op on Windows: the permission bits it would check do not
// exist there. Access is governed by ACLs, os.Chmod toggles only the read-only
// attribute, and fi.Mode().Perm() returns a synthesised 0666/0444 that nothing
// set — so a 0600 comparison would report FAILED on every healthy machine.
//
// This is "not applicable", not "not checked": there is no Windows property this
// declaration describes. The remaining checks — the file exists, is a regular
// file, and is non-empty — do run, and they are the ones that carry meaning on
// both platforms.
//
// If the ACL equivalent is ever wanted (deny-all except the owner), it is a real
// check with a real subject and belongs here as one, not as a reinterpretation of
// a number Windows does not keep.
func checkKeyMode(_ os.FileInfo, _ string, _ fs.FileMode) error { return nil }
