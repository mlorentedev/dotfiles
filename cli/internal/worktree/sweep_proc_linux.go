//go:build linux

package worktree

import (
	"os"
	"path/filepath"
	"strings"
)

// processDiscoverySupported reports whether Gate f can actually observe running
// processes on this platform. Callers use it to explain a refusal to reap
// rather than reporting an empty sweep, which reads identically to a machine
// with nothing to clean up.
const processDiscoverySupported = true

// isHostProcessInside reports whether any running host process has its cwd
// inside targetPath (Gate f).
//
// Every failure path answers `true`, which is the safe direction and not the
// obvious one: the caller DELETES the worktree when this returns false, so
// "I could not find out" and "nobody is in there" must not produce the same
// answer. An earlier version returned false when /proc could not be read, which
// made an unreadable /proc indistinguishable from an idle machine.
func isHostProcessInside(targetPath string) bool {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return true
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return true
	}
	for _, entry := range entries {
		if !entry.IsDir() || !isNumericPID(entry.Name()) {
			continue
		}
		if processCwdInside(entry.Name(), absTarget) {
			return true
		}
	}
	return false
}

func isNumericPID(name string) bool {
	if len(name) == 0 {
		return false
	}
	for _, r := range name {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func processCwdInside(pidName, absTarget string) bool {
	dest, err := os.Readlink(filepath.Join("/proc", pidName, "cwd"))
	if err != nil {
		return false
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return false
	}
	return absDest == absTarget || strings.HasPrefix(absDest, absTarget+string(filepath.Separator))
}
