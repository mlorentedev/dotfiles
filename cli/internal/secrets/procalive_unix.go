//go:build !windows

package secrets

import (
	"errors"
	"syscall"
)

// ProcessAlive reports whether pid names a live process. It is the liveness
// half of the bw serve trace (#1315): a pid file with nothing listening on the
// port means "the daemon died" only if the pid is gone, and "still starting,
// or something else wearing that pid" if it is not.
//
// Signal 0 performs the permission check and delivers nothing. EPERM means the
// process exists but belongs to another user — alive, for this question. A
// killed-but-unreaped child is still alive to kill(2); the daemon is spawned
// under Setsid by a dotf that exits, so init reaps it and the case does not
// arise in production. Windows equivalent in procalive_windows.go.
func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	return errors.Is(err, syscall.EPERM)
}
