//go:build windows

package filelock

import (
	"errors"
	"syscall"
)

const errSharingViolation syscall.Errno = 32

// TryLock takes the lock on path without waiting, or returns ErrLocked. With
// dwShareMode 0 no other open of the file succeeds while the handle is open, so
// the handle itself is the lock and closing it is the unlock.
func TryLock(path string) (func(), error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(p,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0, // dwShareMode 0: exclusive
		nil,
		syscall.OPEN_ALWAYS,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		if errors.Is(err, errSharingViolation) {
			return nil, ErrLocked
		}
		return nil, err
	}
	return func() { _ = syscall.CloseHandle(h) }, nil
}
