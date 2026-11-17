//go:build windows

package converge

import "golang.org/x/sys/windows"

// processElevated reports whether dotf runs with an elevated token, which is
// what a machine-wide winget upgrade needs. Windows has no euid: os.Geteuid
// answers -1 there, so the unix test would call every run unprivileged.
func processElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
