//go:build darwin

package worktree

import "os"

// isCallerInside walks from the caller's parent up the process tree and
// reports the first process whose working directory is target or below it,
// as the Linux walk does. An ancestor whose cwd cannot be read is skipped,
// and a table that cannot be read answers "not inside": a done that refused
// every call would leave no way to remove a worktree (done_proc_other.go).
func isCallerInside(target string) (ancestor, bool) {
	procs, err := readProcTable()
	if err != nil {
		return ancestor{}, false
	}
	pid := os.Getppid()
	for depth := 0; pid > 1 && depth < maxAncestorDepth; depth++ {
		p, ok := procs[pid]
		if !ok {
			break
		}
		if p.Cwd != "" && pathWithin(p.Cwd, target) {
			return ancestor{PID: pid, Comm: p.Comm, Cwd: p.Cwd}, true
		}
		pid = p.PPID
	}
	return ancestor{}, false
}
