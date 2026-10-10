//go:build linux

package worktree

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// isCallerInside walks from the caller's parent up to init and reports the
// first process whose working directory is target or below it.
//
// An ancestor whose cwd cannot be read is SKIPPED, which is the opposite of
// Gate f in sweep_proc_linux.go, and the difference is in the question. Gate f
// asks "is anyone at all inside?" over every process on the machine, so a
// process it cannot read could be the one inside. This asks "did I come from
// there?" over a short fixed chain, and the members an ordinary user cannot
// read are root's: init and, under ssh or a display manager, the daemons that
// spawned the session. Refusing on them would make done refuse on every Linux
// box, since /proc/1/cwd is unreadable to every non-root caller.
func isCallerInside(target string) (ancestor, bool) {
	pid := os.Getppid()
	for depth := 0; pid > 1 && depth < maxAncestorDepth; depth++ {
		proc := filepath.Join("/proc", strconv.Itoa(pid))
		if cwd, err := os.Readlink(filepath.Join(proc, "cwd")); err == nil && pathWithin(cwd, target) {
			comm, _ := os.ReadFile(filepath.Join(proc, "comm")) //nolint:gosec // a fixed /proc path
			return ancestor{PID: pid, Comm: strings.TrimSpace(string(comm)), Cwd: cwd}, true
		}
		pid = parentPID(proc)
	}
	return ancestor{}, false
}

// parentPID reads the ppid from <proc>/stat, or 0 to stop the walk. The second
// field is the command name in parentheses and may itself hold spaces and
// parentheses, so the fields are counted from the LAST closing one.
func parentPID(proc string) int {
	stat, err := os.ReadFile(filepath.Join(proc, "stat")) //nolint:gosec // a fixed /proc path
	if err != nil {
		return 0
	}
	s := string(stat)
	end := strings.LastIndex(s, ")")
	if end < 0 {
		return 0
	}
	fields := strings.Fields(s[end+1:])
	if len(fields) < 2 {
		return 0
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0
	}
	return ppid
}
