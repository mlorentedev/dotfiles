//go:build !linux

package worktree

// processDiscoverySupported is false wherever Gate f has no implementation, so
// a caller can say why nothing was reaped instead of reporting an empty sweep.
const processDiscoverySupported = false

// isHostProcessInside answers `Inside: true` unconditionally off Linux, because
// there is no implementation here and the caller deletes the worktree on a
// false.
//
// This is a real gap, stated rather than hidden. Gate f previously read /proc
// with no build-tag split at all, and `os.ReadDir("/proc")` returning an error
// was treated as "no process is inside" -- so on Windows, where /proc never
// exists, the gate was silently OFF on every run and a worktree containing a
// live shell classified as reapable. The rest of this package already crosses
// the platform boundary properly (lock_unix.go / lock_windows.go); this gate
// did not, and the divergence was invisible because the Linux CI leg exercises
// the only path that works.
//
// Uninspectable stays zero: nothing was inspected, and reporting a count of
// processes we failed to read would imply a scan happened. The absence of the
// scan is carried by processDiscoverySupported instead.
//
// Until a real implementation lands -- ToolHelp32Snapshot plus per-process cwd
// on Windows, libproc on Darwin -- refusing to reap is the correct behaviour:
// an inert sweep costs disk, a wrong one costs uncommitted work.
func isHostProcessInside(_ string) GateFReading {
	return GateFReading{Inside: true}
}
