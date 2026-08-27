//go:build windows

package secrets

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

// TestHelperConsoleProbe is the child half of
// TestBWServeDetachAttr_ChildHasNoConsole. Re-executed as a helper process, it
// reports whether it owns a console: GetConsoleWindow returns NULL for a
// process created with DETACHED_PROCESS and a window handle for one attached
// to its parent's console. Skipped (not failed) when run as an ordinary test.
func TestHelperConsoleProbe(t *testing.T) {
	if os.Getenv("DOTF_CONSOLE_PROBE") != "1" {
		t.Skip("helper process only")
	}
	hwnd, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow").Call()
	if hwnd == 0 {
		_, _ = os.Stdout.WriteString("console=none\n")
	} else {
		_, _ = os.Stdout.WriteString("console=attached\n")
	}
	os.Exit(0)
}

func probeConsole(t *testing.T, attr *syscall.SysProcAttr) string {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperConsoleProbe$") //nolint:gosec // re-exec of the test binary itself
	cmd.Env = append(os.Environ(), "DOTF_CONSOLE_PROBE=1")
	cmd.SysProcAttr = attr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("helper process: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// The property WIN-012/#1293 needs, asserted by effect rather than by reading
// the flag back: a child spawned with bwServeDetachAttr() owns no console, so a
// closing terminal has no CTRL_CLOSE_EVENT to deliver it. The control spawn
// (no attr) proves the environment can tell the two states apart — where the
// test process itself has no console both probes read "none" and nothing is
// proven, so the test says so instead of passing vacuously.
func TestBWServeDetachAttr_ChildHasNoConsole(t *testing.T) {
	if got := probeConsole(t, nil); got != "console=attached" {
		t.Skipf("test process has no console (control probe: %q); detachment cannot be observed here", got)
	}
	if got := probeConsole(t, bwServeDetachAttr()); got != "console=none" {
		t.Fatalf("child spawned with bwServeDetachAttr() still owns a console (%q): it dies with the terminal that started it", got)
	}
}
