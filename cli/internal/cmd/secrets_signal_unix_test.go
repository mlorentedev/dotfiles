//go:build !windows

package cmd

import (
	"bytes"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// CLI-042 AC7 makes `dotf secrets run` a daemon supervisor (`hive serve` under
// systemd), so its signal contract stops being academic: a SIGTERM that kills
// `dotf` without reaching the child leaves the daemon orphaned and the unit
// lying about what it stopped.
//
// The test runs on a NESTED helper rather than signalling the test binary
// itself. Signalling ourselves would work — but against a regressed runChild the
// unhandled SIGTERM would kill the test RUNNER, taking the whole package down
// instead of reporting one red test. A guard whose failure mode is "the harness
// dies" trains people to disable it. Here the death is contained in a
// subprocess, so a regression reports as an ordinary assertion failure.
//
// Layering: test -> forwarder helper (calls runChild, signals itself) -> trap
// helper (catches SIGTERM, exits 42).

const (
	trapChildExitCode = 42
	helperDeadline    = 10 * time.Second
)

// TestTrapHelperProcess is not a real test. Re-exec'd with GO_HELPER_MODE=trap,
// it installs a SIGTERM handler, announces readiness by creating READY_FILE, and
// exits 42 when signalled. The marker is what makes the parent's signal
// deterministic rather than racing a sleep: a SIGTERM delivered before
// signal.Notify runs would kill it outright and the test would read as a
// forwarding failure it did not have.
func TestTrapHelperProcess(t *testing.T) {
	if os.Getenv("GO_HELPER_MODE") != "trap" {
		return
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	if f := os.Getenv("READY_FILE"); f != "" {
		if err := os.WriteFile(f, []byte("ready"), 0o600); err != nil {
			os.Exit(90) // distinct from every expected code, so a setup failure is not read as a verdict
		}
	}
	select {
	case <-sigs:
		os.Exit(trapChildExitCode)
	case <-time.After(helperDeadline):
		os.Exit(91) // never signalled — the forwarding under test did not happen
	}
}

// TestForwarderHelperProcess is not a real test. Re-exec'd with
// GO_HELPER_MODE=forwarder it plays the role `dotf` plays under systemd: it
// calls runChild on the trap helper and then sends itself a SIGTERM, exactly as
// a supervisor would. It prints the child's exit code so the parent can assert
// on it.
//
// A regressed runChild (no signal.Notify) dies here on the unhandled SIGTERM,
// and the parent observes a signal-terminated helper with no output.
func TestForwarderHelperProcess(t *testing.T) {
	if os.Getenv("GO_HELPER_MODE") != "forwarder" {
		return
	}
	ready := os.Getenv("READY_FILE")
	go func() {
		deadline := time.Now().Add(helperDeadline)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(ready); err == nil {
				_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	environ := append(os.Environ(), "GO_HELPER_MODE=trap")
	code, err := runChild(
		[]string{os.Args[0], "-test.run=TestTrapHelperProcess"},
		environ, nil, io.Discard, io.Discard,
	)
	if err != nil {
		os.Exit(92)
	}
	_, _ = io.WriteString(os.Stdout, "CHILD_EXIT="+strconv.Itoa(code))
	os.Exit(0)
}

func TestRunChild_ForwardsSIGTERMToChild(t *testing.T) {
	ready := filepath.Join(t.TempDir(), "ready")

	var out bytes.Buffer
	environ := append(os.Environ(), "GO_HELPER_MODE=forwarder", "READY_FILE="+ready)
	code, err := runChild(
		[]string{os.Args[0], "-test.run=TestForwarderHelperProcess"},
		environ, nil, &out, io.Discard,
	)
	if err != nil {
		t.Fatalf("running the forwarder helper: %v", err)
	}
	if code != 0 {
		t.Fatalf("forwarder helper exited %d (want 0); a signal-terminated or "+
			"non-zero helper means runChild did not forward SIGTERM and was "+
			"killed by it instead. stdout=%q", code, out.String())
	}
	want := "CHILD_EXIT=" + strconv.Itoa(trapChildExitCode)
	if !strings.Contains(out.String(), want) {
		t.Errorf("forwarder reported %q, want %q — the trap child did not "+
			"receive the forwarded SIGTERM", out.String(), want)
	}
}
