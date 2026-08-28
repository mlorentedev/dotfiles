package secrets

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestHelperFakeDaemon is the child half of the process-level tests here. Re-
// executed as a helper process it prints one line on each stream and then
// sleeps until killed — the shape of a daemon: something that starts, says a
// little, and stays. Skipped (not failed) when run as an ordinary test.
func TestHelperFakeDaemon(t *testing.T) {
	if os.Getenv("DOTF_FAKE_DAEMON") != "1" {
		t.Skip("helper process only")
	}
	_, _ = fmt.Fprintln(os.Stdout, "fake daemon: stdout line")
	_, _ = fmt.Fprintln(os.Stderr, "fake daemon: stderr line")
	time.Sleep(60 * time.Second)
	os.Exit(0)
}

// fakeDaemonCmd builds a command that re-executes the test binary as
// TestHelperFakeDaemon, carrying the PRODUCTION detach attributes: on Windows
// that is DETACHED_PROCESS, which is precisely the combination AC4 has to
// prove (a detached child with redirected stdio still starts).
func fakeDaemonCmd(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperFakeDaemon$") //nolint:gosec // re-exec of the test binary itself
	cmd.Env = append(os.Environ(), "DOTF_FAKE_DAEMON=1")
	cmd.SysProcAttr = bwServeDetachAttr()
	return cmd
}

// reap kills a helper and waits for it, so a killed pid is also a reaped one
// — the state ProcessAlive is specified against.
func reap(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

// TestProcessAlive is AC6 by effect: a running child is alive, the same pid is
// not once killed and reaped. Both halves matter — a probe that always said
// "alive" would pass the first and a probe that always said "dead" the second.
func TestProcessAlive(t *testing.T) {
	cmd := fakeDaemonCmd(t)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	pid := cmd.Process.Pid
	if !ProcessAlive(pid) {
		reap(t, cmd)
		t.Fatalf("ProcessAlive(%d) = false for a running child", pid)
	}
	reap(t, cmd)
	if ProcessAlive(pid) {
		t.Fatalf("ProcessAlive(%d) = true after the child was killed and reaped", pid)
	}
}

func TestProcessAlive_NonPositivePIDIsDead(t *testing.T) {
	for _, pid := range []int{0, -1} {
		if ProcessAlive(pid) {
			t.Fatalf("ProcessAlive(%d) must be false", pid)
		}
	}
}

func TestProcessAlive_SelfIsAlive(t *testing.T) {
	if !ProcessAlive(os.Getpid()) {
		t.Fatal("ProcessAlive(own pid) = false")
	}
}
