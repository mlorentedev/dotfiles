//go:build windows

package secrets

import "syscall"

// detachedProcess is Win32's DETACHED_PROCESS creation flag. The syscall
// package exports CREATE_NEW_PROCESS_GROUP but not this one; golang.org/x/sys
// does, but promoting it to a direct dependency for a single constant is not
// worth the diff — the value is documented and fixed.
const detachedProcess = 0x00000008

// bwServeDetachAttr detaches the daemon from this process's console so it
// outlives the terminal that started it — the whole point of one unlock
// serving every later `dotf` call (bwserve.go). Two flags, two different
// properties, and only together do they give the Windows analogue of
// bwserve_unix.go's Setsid:
//
//   - CREATE_NEW_PROCESS_GROUP re-routes Ctrl+C so a Ctrl+C to the CLI does
//     not also hit the daemon. It does NOT detach from the console.
//   - DETACHED_PROCESS creates the child with no console at all. Without it a
//     console-subsystem child stays attached to its parent's console, and when
//     that console closes Windows delivers CTRL_CLOSE_EVENT and terminates the
//     child — measured on the Windows work box (WIN-012/#1293): unlock in one
//     terminal, close it, and every wrapper in every other terminal failed at
//     once with "no bw serve daemon is running". The CLI exiting was never the
//     problem (stdio is NUL, and Windows has no parent→child kill); the
//     terminal closing was.
//
// TestBWServeDetachAttr_ChildHasNoConsole asserts the property by effect.
func bwServeDetachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess}
}
