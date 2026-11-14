//go:build !windows

package lane

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Exec replaces this process with argv. The agent then owns the terminal
// directly: no dotf process stays between it and the pane, so its signals,
// resize events and exit status are the agent's own. A launch through the
// secrets wrapper starts with the agent's name in HERDR_AGENT, so herdr can name
// the pane (LaunchEnv, #3430).
func Exec(argv []string) error {
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		return fmt.Errorf("lane: %s is not on PATH: %w", argv[0], err)
	}
	return syscall.Exec(bin, argv, withEnv(os.Environ(), LaunchEnv(argv))) //nolint:gosec // argv comes from the launch table, bin from PATH
}
