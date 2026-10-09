package secrets

import (
	"io/fs"
	"os"
)

// declaredKeyMode is the mode a file-authority root must carry: its
// declaration's, else 0600.
func declaredKeyMode(want fs.FileMode) fs.FileMode {
	if want == 0 {
		return 0o600
	}
	return want.Perm()
}

// CheckFileAuthorityMode reports whether the root at e.Dest carries its declared
// permission bits. It is nil where the OS keeps no such bits (see checkKeyMode's
// two halves), so a caller never has to ask which OS it runs on.
func CheckFileAuthorityMode(e Entry) error {
	fi, err := os.Stat(e.Dest)
	if err != nil {
		return err
	}
	return checkKeyMode(fi, e.Dest, e.Mode)
}

// RepairFileAuthorityMode sets the root's declared permission bits. Only the
// mode changes; the key's content is never read.
func RepairFileAuthorityMode(e Entry) error {
	return os.Chmod(e.Dest, declaredKeyMode(e.Mode))
}
