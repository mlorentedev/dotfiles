//go:build darwin

package bridge

import (
	"bytes"
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

func observeProcess(pid int) (processSnapshot, error) {
	if pid <= 0 {
		return processSnapshot{}, errProcessNotFound
	}
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		if errors.Is(err, unix.ESRCH) || errors.Is(err, unix.ENOENT) {
			return processSnapshot{}, errProcessNotFound
		}
		return processSnapshot{}, err
	}
	if int(info.Proc.P_pid) != pid {
		return processSnapshot{}, errProcessNotFound
	}
	executable, err := darwinProcessPath(pid)
	if err != nil {
		return processSnapshot{}, err
	}
	start := info.Proc.P_starttime
	return processSnapshot{
		Executable:    executable,
		StartIdentity: fmt.Sprintf("darwin:%d:%d", start.Sec, start.Usec),
	}, nil
}

func darwinProcessPath(pid int) (string, error) {
	raw, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return "", err
	}
	if len(raw) <= 4 {
		return "", errors.New("process arguments do not contain an executable path")
	}
	path := raw[4:]
	if end := bytes.IndexByte(path, 0); end >= 0 {
		path = path[:end]
	}
	if len(path) == 0 {
		return "", errors.New("process executable path is empty")
	}
	return string(path), nil
}
