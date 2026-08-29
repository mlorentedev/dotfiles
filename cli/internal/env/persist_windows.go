//go:build windows

package env

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// registryUserEnv is the production UserEnvStore: HKCU\Environment, the scope
// [Environment]::SetEnvironmentVariable(name, value, 'User') writes and every
// new process reads at start — including the profile-less ones.
type registryUserEnv struct{}

// NewUserEnvStore returns the per-user persistent environment on Windows.
func NewUserEnvStore() (UserEnvStore, error) { return registryUserEnv{}, nil }

func (registryUserEnv) Get(name string) (string, bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = k.Close() }()
	v, _, err := k.GetStringValue(name)
	if errors.Is(err, registry.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func (registryUserEnv) Set(name, value string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	if err := k.SetStringValue(name, value); err != nil {
		return err
	}
	broadcastEnvironmentChange()
	return nil
}

// broadcastEnvironmentChange tells running shells and Explorer that the user
// environment changed (what setx does after writing the same key), so a
// terminal opened from the taskbar afterwards sees the value without a
// logoff. Best effort: a failure here leaves the registry correct and only
// delays visibility to the next logon.
func broadcastEnvironmentChange() {
	const (
		hwndBroadcast   = uintptr(0xffff)
		wmSettingChange = uintptr(0x001A)
		smtoAbortIfHung = uintptr(0x0002)
	)
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("SendMessageTimeoutW")
	lparam, err := syscall.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	var result uintptr
	_, _, _ = proc.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(lparam)),
		smtoAbortIfHung, 5000, uintptr(unsafe.Pointer(&result)))
}
