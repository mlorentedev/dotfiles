//go:build windows

package worktree

import (
	"errors"
	"syscall"
)

const errSharingViolation syscall.Errno = 32

func TryLockFile(path string) (func(), error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(
		p,
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

	unlock := func() {
		_ = syscall.CloseHandle(h)
	}
	return unlock, nil
}
