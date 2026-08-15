//go:build !windows

package secrets

import "syscall"

// bwServeDetachAttr detaches the daemon from this process's session (Setsid)
// so it survives the CLI invocation exiting. POSIX-only field — see
// bwserve_windows.go for the Windows equivalent.
func bwServeDetachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
