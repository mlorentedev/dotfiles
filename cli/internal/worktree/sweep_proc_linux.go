//go:build linux

package worktree

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// processDiscoverySupported reports whether Gate f can actually observe running
// processes on this platform. Callers use it to explain a refusal to reap
// rather than reporting an empty sweep, which reads identically to a machine
// with nothing to clean up.
const processDiscoverySupported = true

// isHostProcessInside reports what Gate f can establish about targetPath.
//
// Every failure path that prevents the scan from happening answers
// `Inside: true`, which is the safe direction and not the obvious one: the
// caller DELETES the worktree on a false, so "I could not find out" and "nobody
// is in there" must not produce the same answer. An earlier version returned
// false when /proc could not be read, which made an unreadable /proc
// indistinguishable from an idle machine.
//
// The target is resolved through EvalSymlinks before comparing, because
// /proc/<pid>/cwd is ALREADY resolved by the kernel and filepath.Abs is not.
// Without it a worktree reached by a symlinked path compares unequal to the
// physical path every process reports, so the gate answers "nobody is inside"
// while a shell sits in it — the same fail-open this file exists to remove,
// arriving through a different door.
func isHostProcessInside(targetPath string) GateFReading {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return GateFReading{Inside: true}
	}
	resolved, err := filepath.EvalSymlinks(absTarget)
	if err != nil {
		// The caller is about to delete this path, so it should exist. If it
		// cannot be resolved, the comparison below cannot be trusted either.
		return GateFReading{Inside: true}
	}
	absTarget = resolved

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return GateFReading{Inside: true}
	}

	reading := GateFReading{}
	for _, entry := range entries {
		if !entry.IsDir() || !isNumericPID(entry.Name()) {
			continue
		}
		switch inspectProcessCwd(entry.Name(), absTarget) {
		case cwdInside:
			reading.Inside = true
			return reading
		case cwdUnreadable:
			reading.Uninspectable++
		case cwdOutside:
		}
	}
	return reading
}

// cwdVerdict is three-valued on purpose. "Not inside" and "I was not allowed to
// look" are different facts, and collapsing them into one boolean is what the
// review caught: a root process sitting in a worktree reads exactly like an
// empty one.
type cwdVerdict int

const (
	cwdOutside cwdVerdict = iota
	cwdInside
	cwdUnreadable
)

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

// inspectProcessCwd answers for one process.
//
// A vanished process is `cwdOutside` and not `cwdUnreadable`: /proc is a
// snapshot, processes exit between the ReadDir and the Readlink constantly, and
// a process that no longer exists is genuinely not in the worktree.
//
// EACCES is a real limitation and is reported rather than resolved.
// /proc/<pid>/cwd is readable only by the process owner and root, so a scan run
// as an ordinary user cannot see another user's cwd — including root's. Making
// that answer `Inside: true` would be the fail-closed reflex and it would make
// sweep permanently inert on Linux, since /proc/1/cwd is EACCES to every
// non-root caller and therefore every scan would refuse. Counting them instead
// keeps the tool useful while making a partial scan visible as a partial scan,
// which is the same choice ProcessDiscovery makes one level up.
func inspectProcessCwd(pidName, absTarget string) cwdVerdict {
	dest, err := os.Readlink(filepath.Join("/proc", pidName, "cwd"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cwdOutside
		}
		return cwdUnreadable
	}
	// Already absolute and already symlink-resolved by the kernel; Abs only
	// guards against a relative answer from an unexpected kernel.
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return cwdUnreadable
	}
	if absDest == absTarget || strings.HasPrefix(absDest, absTarget+string(filepath.Separator)) {
		return cwdInside
	}
	return cwdOutside
}
