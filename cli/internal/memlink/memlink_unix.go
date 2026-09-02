//go:build !windows

package memlink

import "os"

// createLink makes a POSIX symlink. No cmd.exe-style tokenizer sits between
// the caller and the syscall here, so no quoting dance is needed — see
// memlink_windows.go for the Windows counterpart and why it needs one.
func createLink(src, target string) error {
	return os.Symlink(src, target)
}
