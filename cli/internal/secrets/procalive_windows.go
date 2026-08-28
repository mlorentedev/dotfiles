//go:build windows

package secrets

import (
	"errors"
	"syscall"
)

// Win32 constants the syscall package does not export. Kept local like
// detachedProcess (bwserve_windows.go): promoting golang.org/x/sys to a direct
// dependency for two documented, fixed values is not worth the diff.
const (
	// processQueryLimitedInformation is the least access that lets
	// GetExitCodeProcess answer, and it is granted on processes of a higher
	// integrity level where PROCESS_QUERY_INFORMATION is refused.
	processQueryLimitedInformation = 0x1000
	// stillActive is the exit code GetExitCodeProcess reports for a process
	// that has not exited.
	stillActive = 259
)

// ProcessAlive reports whether pid names a live process — the Windows half of
// the bw serve trace (#1315); see procalive_unix.go for the question it
// answers. os.FindProcess is not enough here: on Windows it succeeds for any
// pid that opens, exited or not, so the exit code has to be read.
//
// ERROR_ACCESS_DENIED from OpenProcess means the process exists but cannot be
// opened from this token — alive, the same reading Unix gives EPERM.
func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid)) //nolint:gosec // pid > 0 checked above
	if err != nil {
		return errors.Is(err, syscall.ERROR_ACCESS_DENIED)
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == stillActive
}
