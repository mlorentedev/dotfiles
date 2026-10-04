// Package claude converges the parts of a Claude Code install that are not
// config files: the plugins `claude plugin install` adds, and the snapshot
// guard every claude CLI call needs around it (CLI-063, #1339).
//
// It is the Go side of a port. setup-linux.sh and setup-windows.ps1 still do
// the same work until the cutover, and tests/claude-plugins.bats keeps the
// three plugin lists equal while they coexist.
package claude

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Truncated reports whether a .claude.json that was snapshot bytes before a
// claude CLI call and is now bytes after it was cut down by the upstream strip
// bug (anthropics/claude-code#59870): the snapshot was at least floor bytes
// and the file lost more than half of it.
//
// "More than half" is exact: 2*now < snapshot. The Linux twin compares against
// an integer-divided half and so misses an odd-sized snapshot cut to exactly
// floor(snapshot/2); the PowerShell twin compares against the exact half. Go
// takes the exact one (recorded in the spec's divergences.md).
func Truncated(snapshot, now, floor int64) bool {
	return snapshot >= floor && 2*now < snapshot
}

// Guard runs fn with path snapshotted and puts the snapshot back if fn left the
// file truncated. It reports whether it restored, and returns fn's error; a
// failed install truncates just as well as a successful one.
//
// The snapshot is held in memory and restored by rewriting the file in place,
// as both twins do with cp/Copy-Item. Rewriting in place keeps the file's own
// permissions, which matters on Windows, where .claude.json carries the
// session's OAuth state and a new file would inherit the directory's ACL.
func Guard(path string, floor int, fn func() error) (bool, error) {
	snapshot, err := os.ReadFile(path) //nolint:gosec // the user's own claude config
	if errors.Is(err, fs.ErrNotExist) {
		return false, fn()
	}
	if err != nil {
		return false, fmt.Errorf("snapshot %s: %w", path, err)
	}
	runErr := fn()
	info, err := os.Stat(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, errors.Join(runErr, fmt.Errorf("stat %s: %w", path, err))
	}
	size := int64(0)
	if err == nil {
		size = info.Size()
	}
	if !Truncated(int64(len(snapshot)), size, int64(floor)) {
		return false, runErr
	}
	if err := os.WriteFile(path, snapshot, 0o600); err != nil {
		return false, errors.Join(runErr, fmt.Errorf("restore %s: %w", path, err))
	}
	return true, runErr
}
