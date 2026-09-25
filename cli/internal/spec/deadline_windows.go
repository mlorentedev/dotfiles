//go:build windows

package spec

import (
	"os"
	"os/exec"
	"time"
)

// stopSignals are the signals that end a run the way the deadline does.
var stopSignals = []os.Signal{os.Interrupt}

// startInOwnGroup is a no-op on Windows, where process groups do not carry
// signals. A review there runs in the foreground, in the operator's terminal.
func startInOwnGroup(*exec.Cmd) {}

// stopGroup kills the runner; on Windows its children are not reached.
func stopGroup(c *exec.Cmd, _ time.Duration, done <-chan error) {
	_ = c.Process.Kill()
	<-done
}
