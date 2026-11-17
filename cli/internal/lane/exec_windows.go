//go:build windows

package lane

import (
	"fmt"
	"os"
	"os/exec"
)

// nativeBehind is where Exec asks for the executable behind a batch shim. A
// variable so a test answers without a mise.
var nativeBehind = MiseNative

// Exec runs argv as a child on this console and waits: Windows cannot replace a
// process image, so dotf stays as the parent and hands the agent the same
// stdio. A .cmd or .bat target is never started: cmd.exe would re-parse the
// prompt (RefuseBatchShim, #3429). The native executable the tool manager holds
// behind such a shim is started in its place, and without one the launch is
// refused (NativeTarget, #3569).
func Exec(argv []string) error {
	if _, err := exec.LookPath(argv[0]); err != nil {
		return fmt.Errorf("lane: %s is not on PATH: %w", argv[0], err)
	}
	bin, err := NativeTarget(argv[0], exec.LookPath, nativeBehind)
	if err != nil {
		return fmt.Errorf("lane: %w", err)
	}
	cmd := exec.Command(bin, argv[1:]...) //nolint:gosec // argv comes from the launch table, bin from PATH
	cmd.Env = withEnv(os.Environ(), LaunchEnv(argv))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
