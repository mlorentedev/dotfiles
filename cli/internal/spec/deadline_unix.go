//go:build !windows

package spec

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

// stopSignals are the signals that end a run the way the deadline does.
var stopSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}

// startInOwnGroup puts the runner in a process group of its own, so the
// deadline can stop everything it starts with one signal.
func startInOwnGroup(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// stopGroup asks the runner's group to stop, then kills it after grace, and
// waits for the runner to be reaped either way.
func stopGroup(c *exec.Cmd, grace time.Duration, done <-chan error) {
	pgid := -c.Process.Pid
	_ = syscall.Kill(pgid, syscall.SIGTERM)
	select {
	case <-done:
		// The runner is gone. Its children may still be running.
		_ = syscall.Kill(pgid, syscall.SIGKILL)
	case <-time.After(grace):
		_ = syscall.Kill(pgid, syscall.SIGKILL)
		<-done
	}
}
