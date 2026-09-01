// Package fsmode applies a declared file mode where the OS can actually
// express it (CLI-055, #1302).
//
// The manifest and the secrets registry declare modes in POSIX octal — `0600`
// for a rendered credential — and every writer reached disk through os.Chmod.
// On Windows os.Chmod toggles the read-only attribute and nothing else, so a
// file declared 0600 was protected only by whatever ACL its directory handed
// down. Apply keeps os.Chmod's behaviour everywhere and, on Windows, turns an
// owner-only mode into an owner-only DACL. One place decides what a mode
// means; the writers only say which mode.
package fsmode

import "os"

// Apply sets mode on path: os.Chmod on every OS, and on Windows an owner-only
// DACL as well when mode grants nothing to group or other.
func Apply(path string, mode os.FileMode) error {
	if err := os.Chmod(path, mode); err != nil {
		return err
	}
	if !OwnerOnly(mode) {
		return nil
	}
	return restrictToOwner(path)
}

// OwnerOnly reports whether mode grants nothing to group or other — the modes
// (0600, 0700, 0400) that mean "this file is mine alone".
func OwnerOnly(mode os.FileMode) bool {
	return mode.Perm()&0o077 == 0
}

// Needs reports whether Apply(path, mode) would change anything: the POSIX
// bits differ, or — on Windows, for an owner-only mode — the file's DACL is
// not yet protected. It is the read-only question a deploy asks about a file
// whose content is already in sync, so a mode an older binary could not
// express gets applied on the next run instead of never (CLI-055).
func Needs(path string, mode os.FileMode) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if permDiffers(fi.Mode(), mode) {
		return true, nil
	}
	if !OwnerOnly(mode) {
		return false, nil
	}
	applied, err := ownerOnlyApplied(path)
	if err != nil {
		return false, err
	}
	return !applied, nil
}

