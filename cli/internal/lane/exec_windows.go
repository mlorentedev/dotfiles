//go:build windows

package lane

import (
	"fmt"
	"os"
	"os/exec"
)

// Exec runs argv as a child on this console and waits: Windows cannot replace a
// process image, so dotf stays as the parent and hands the agent the same
// stdio.
func Exec(argv []string) error {
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		return fmt.Errorf("lane: %s is not on PATH: %w", argv[0], err)
	}
	cmd := exec.Command(bin, argv[1:]...) //nolint:gosec // argv comes from the launch table, bin from PATH
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
