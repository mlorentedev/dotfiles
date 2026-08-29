//go:build !windows

package fsmode

import "os"

// restrictToOwner is a no-op off Windows: os.Chmod already expressed the mode
// in full, and there is no second permission system to reconcile.
func restrictToOwner(string) error { return nil }

// ownerOnlyApplied is true off Windows for the same reason: once the bits are
// on, nothing else stands between the file and "mine alone".
func ownerOnlyApplied(string) (bool, error) { return true, nil }

// permDiffers compares the full permission bits: POSIX expresses all of them.
func permDiffers(got, want os.FileMode) bool { return got.Perm() != want.Perm() }
