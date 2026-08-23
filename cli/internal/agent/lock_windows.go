//go:build windows

package agent

import (
	"errors"
	"os"
	"syscall"
)

// ERROR_SHARING_VIOLATION — another process already holds the file with a share
// mode that excludes us. Declared here because the `syscall` package does not
// export it on Windows.
const errSharingViolation syscall.Errno = 32

// tryTakeSlot opens the slot file with an exclusive share mode, which IS the
// lock on Windows: while this handle is open, no other process can open the
// same path. It reports (nil, false, nil) when someone already holds it.
//
// This is the Windows counterpart of flock and gives the property the semaphore
// depends on — the handle belongs to the process, so the OS closes it and frees
// the slot when the process dies, however it dies.
//
// The share-mode route is used rather than LockFileEx because `syscall` does
// not export LockFileEx on Windows; reaching it would mean promoting
// `golang.org/x/sys` from an indirect dependency to a direct one, which is
// itself a Discipline Gate trigger. It is also the reason this file exists at
// all: the Windows leg of CI compiles this same tree, and a unix-only syscall
// here passes every local check while failing the whole package there (#1075).
func tryTakeSlot(path string) (*os.File, bool, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, false, err
	}
	h, err := syscall.CreateFile(
		p,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0, // dwShareMode 0: no other process may open this file — the lock itself
		nil,
		syscall.OPEN_ALWAYS,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err == nil {
		return os.NewFile(uintptr(h), path), true, nil
	}
	if errors.Is(err, errSharingViolation) {
		return nil, false, nil
	}
	return nil, false, err
}

// releaseSlot closes the handle, which frees the slot: with dwShareMode 0 the
// open handle IS the lock, so there is nothing else to unlock.
func releaseSlot(f *os.File) {
	_ = f.Close()
}
