//go:build windows

package secrets

import "syscall"

// bwServeDetachAttr detaches the daemon from this process's console group
// (CREATE_NEW_PROCESS_GROUP) so a Ctrl+C to the CLI does not also kill the
// daemon. Windows has no POSIX setsid equivalent, so this is a narrower
// guarantee than bwserve_unix.go's — live Windows daemon-lifecycle behavior
// (surviving the CLI exiting entirely, not just a Ctrl+C) is explicitly
// deferred to empirical validation on Windows (proposal.md's Out of scope),
// not asserted here.
func bwServeDetachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}
