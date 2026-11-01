// Package fsread reads directory trees without filepath.Glob's blind spot:
// Glob drops I/O errors, so an unreadable directory reads as holding nothing
// and a check built on it passes without having looked (lesson 491, #2797).
package fsread

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// ChildFiles is filepath.Glob(root/*/rel) with the errors kept. It returns the
// paths that exist (a directory there counts, as with Glob) in name order, the
// entries whose path could not be checked, and an error when root itself
// cannot be read. A root that does not exist holds nothing, and an entry that
// is not a directory, or has no rel, is skipped.
func ChildFiles(root, rel string) (found []string, unreadable []*fs.PathError, err error) {
	entries, err := os.ReadDir(root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	for _, e := range entries {
		p := filepath.Join(root, e.Name(), rel)
		_, err := os.Lstat(p)
		var pe *fs.PathError
		switch {
		case err == nil:
			found = append(found, p)
		case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
			// not a child dir, or one without rel
		case errors.As(err, &pe):
			unreadable = append(unreadable, pe)
		default:
			unreadable = append(unreadable, &fs.PathError{Op: "lstat", Path: p, Err: err})
		}
	}
	return found, unreadable, nil
}
