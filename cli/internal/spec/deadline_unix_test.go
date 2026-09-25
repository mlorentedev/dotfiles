//go:build !windows

package spec

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// HARNESS-152: the deadline stops the runner AND what it started. A reviewer
// runs as `dotf secrets run -- pi …`, and secrets run starts pi as its child,
// so killing only the direct child would leave pi reviewing, and spending, with
// nobody waiting for it.
func TestRunWithDeadlineStopsARunnerThatOverstaysAndEverythingItStarted(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	argv := []string{"sh", "-c", "sleep 60 & echo $! > " + pidFile + "; wait"}

	began := time.Now()
	code, timedOut, err := RunWithDeadline(time.Second, argv, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if !timedOut || code != ExitDeadline {
		t.Errorf("timedOut=%v code=%d, want true and %d", timedOut, code, ExitDeadline)
	}
	if elapsed := time.Since(began); elapsed > 15*time.Second {
		t.Errorf("the deadline took %s to stop a 1s run", elapsed)
	}

	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if syscall.Kill(pid, 0) == nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("the runner's child (pid %d) outlived the deadline", pid)
	}
}

// A run that ends in time is untouched: its output and its own exit status pass
// through, so a reviewer that fails still reads as failed.
func TestRunWithDeadlinePassesThroughARunnerThatEndsInTime(t *testing.T) {
	var out bytes.Buffer
	code, timedOut, err := RunWithDeadline(10*time.Second, []string{"sh", "-c", "echo reviewed; exit 3"}, nil, &out, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if timedOut || code != 3 {
		t.Errorf("timedOut=%v code=%d, want false and 3", timedOut, code)
	}
	if out.String() != "reviewed\n" {
		t.Errorf("stdout = %q", out.String())
	}
}

// The runner has a process group of its own, so Ctrl-C or `tmux kill-session`
// reaches only this process. It has to pass the signal on, or the reviewer would
// keep running after the session that launched it was killed.
func TestRunWithDeadlinePassesAStopSignalToTheRunnersGroup(t *testing.T) {
	sigs := make(chan os.Signal, 1)
	prev := interrupts
	interrupts = func() (<-chan os.Signal, func()) { return sigs, func() {} }
	t.Cleanup(func() { interrupts = prev })

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	type result struct {
		code     int
		timedOut bool
		err      error
	}
	ran := make(chan result, 1)
	go func() {
		code, timedOut, err := RunWithDeadline(time.Minute,
			[]string{"sh", "-c", "sleep 60 & echo $! > " + pidFile + "; wait"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
		ran <- result{code, timedOut, err}
	}()

	var pid int
	for i := 0; i < 100 && pid == 0; i++ {
		time.Sleep(50 * time.Millisecond)
		if raw, err := os.ReadFile(pidFile); err == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(raw)))
		}
	}
	if pid == 0 {
		t.Fatal("the runner never started its child")
	}
	sigs <- syscall.SIGTERM

	select {
	case r := <-ran:
		if r.err != nil || r.timedOut || r.code != 128+int(syscall.SIGTERM) {
			t.Errorf("code=%d timedOut=%v err=%v, want %d", r.code, r.timedOut, r.err, 128+int(syscall.SIGTERM))
		}
	case <-time.After(15 * time.Second):
		t.Fatal("a stop signal did not end the run")
	}
	time.Sleep(200 * time.Millisecond)
	if syscall.Kill(pid, 0) == nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("the runner's child (pid %d) outlived the stop signal", pid)
	}
}
