//go:build windows

package schedule

import "golang.org/x/sys/windows"

// Elevated reports whether this process runs with an elevated token, which is
// what registering an S4U task needs.
func Elevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
