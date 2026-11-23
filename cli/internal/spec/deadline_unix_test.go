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

// An evaluation run must tell a runner the launcher stopped on a signal from
// one that exited 130 by itself: only the first is not the model's doing. The
// stop reason stays StopNone, so `spec review` reads a signal as it always has.
func TestRunWithLimitsSaysWhenASignalEndedTheRun(t *testing.T) {
	sigs := make(chan os.Signal, 1)
	prev := interrupts
	interrupts = func() (<-chan os.Signal, func()) { return sigs, func() {} }
	t.Cleanup(func() { interrupts = prev })

	started := filepath.Join(t.TempDir(), "started")
	ran := make(chan RunResult, 1)
	go func() {
		res, _ := RunWithLimits(Limits{Deadline: time.Minute}, []string{"sh", "-c", "touch " + started + "; sleep 60"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
		ran <- res
	}()
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(started); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	sigs <- syscall.SIGINT
	select {
	case res := <-ran:
		if !res.Signaled || res.Stop != StopNone {
			t.Errorf("signal: %+v, want Signaled and StopNone", res)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("a stop signal did not end the run")
	}

	res, err := RunWithLimits(Limits{Deadline: time.Minute}, []string{"sh", "-c", "exit 130"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil || res.Signaled || res.Code != 130 {
		t.Errorf("own exit 130: %+v err %v, want not Signaled", res, err)
	}
}

// An evaluation run launches in-process and cannot change this process's
// directory: the runner is started in Limits.Dir, which pi reads as its tree.
func TestRunWithLimitsStartsTheRunnerInItsDirectory(t *testing.T) {
	dir := ResolveExisting(t.TempDir())
	var out bytes.Buffer
	res, err := RunWithLimits(Limits{Deadline: time.Minute, Dir: dir}, []string{"sh", "-c", "pwd -P"}, nil, &out, &bytes.Buffer{})
	if err != nil || res.Code != 0 || strings.TrimSpace(out.String()) != dir {
		t.Errorf("ran in %q (code %d, err %v), want %s", strings.TrimSpace(out.String()), res.Code, err, dir)
	}
}

// os/exec sets PWD to Dir only when it builds the environment itself; an
// explicit Env taken from os.Environ() keeps the launcher's PWD, so a runner
// that reads PWD would see the wrong tree. The filtered environment is built
// from the command's own Environ() for that reason (#3615 review).
//
// printenv, not `sh -c 'echo $PWD'`: a POSIX shell replaces an inherited PWD
// that does not name its directory, so a shell child hides exactly the stale
// value this test exists to catch (it passed against the bug when written so).
func TestRunWithLimitsGivesTheRunnerItsDirectoryAsPWD(t *testing.T) {
	dir := ResolveExisting(t.TempDir())
	var out bytes.Buffer
	res, err := RunWithLimits(Limits{Deadline: time.Minute, Dir: dir}, []string{"printenv", "PWD"}, nil, &out, &bytes.Buffer{})
	if got := strings.TrimSpace(out.String()); err != nil || res.Code != 0 || got != dir {
		t.Errorf("PWD %q (code %d, err %v), want %s", got, res.Code, err, dir)
	}
}
