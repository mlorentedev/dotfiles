//go:build windows

package bridge

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

const stillActive = 259

func observeProcess(pid int) (processSnapshot, error) {
	if pid <= 0 {
		return processSnapshot{}, errProcessNotFound
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return processSnapshot{}, errProcessNotFound
		}
		return processSnapshot{}, err
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return processSnapshot{}, err
	}
	if exitCode != stillActive {
		return processSnapshot{}, errProcessNotFound
	}
	executable, err := processImagePath(handle)
	if err != nil {
		return processSnapshot{}, err
	}
	var created, exited, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return processSnapshot{}, err
	}
	return processSnapshot{
		Executable:    executable,
		StartIdentity: fmt.Sprintf("windows:%d", created.Nanoseconds()),
	}, nil
}

func processImagePath(handle windows.Handle) (string, error) {
	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err != nil {
		return "", err
	}
	if size == 0 {
		return "", errors.New("process executable path is empty")
	}
	return windows.UTF16ToString(buffer[:size]), nil
}
