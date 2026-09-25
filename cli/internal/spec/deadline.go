package spec

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// ExitDeadline is the status of a run the deadline stopped, the one GNU
// timeout uses, so a caller can tell it apart from the runner's own failures.
const ExitDeadline = 124

// deadlineGrace is how long a stopped runner gets to exit on SIGTERM before the
// whole process group is killed.
const deadlineGrace = 10 * time.Second

// TimeBudget is the part of the reviewer's prompt that says when the run ends
// (HARNESS-152).
//
// A reviewer used to learn about the deadline only by hitting it. On pi there
// was none at all, and the SKILL-001 rounds took 31, 39 and more than 63
// minutes. A deadline alone would kill the slow ones with no verdict, wasting
// the whole run, so the reviewer is also given a target: two thirds of the time,
// after which it stops exploring and writes what it verified. The times are
// absolute, because "30 minutes" means nothing to a model that cannot tell when
// the run began, and `date` can.
func TimeBudget(start time.Time, timeout time.Duration) string {
	if timeout <= 0 {
		timeout = DefaultReviewerTimeout
	}
	target := start.Add(timeout * 2 / 3)
	kill := start.Add(timeout)
	const clock = "15:04 MST"
	var b strings.Builder
	b.WriteString("Time budget:\n")
	fmt.Fprintf(&b, "- This run is stopped at %s (`date` shows the time now), %s after launch.\n"+
		"  A stopped run writes no verdict, so the whole run is lost.\n", kill.Format(clock), timeout)
	fmt.Fprintf(&b, "- Aim to finish by %s. From then, stop exploring: write the review with what you\n"+
		"  have verified, and mark what you did not reach as UNVERIFIED rather than leaving no\n"+
		"  verdict at all.\n", target.Format(clock))
	return b.String()
}

// RunWithDeadline runs argv and stops it, with every process it started, when
// timeout passes. It returns the runner's exit status, or ExitDeadline and true
// when the deadline ended the run.
//
// This is how `dotf spec review --timeout` binds every runner (HARNESS-152).
// Before it, only agy was bounded, through its own --print-timeout; pi and any
// runner added later ran for as long as they ran. The whole process group is
// stopped, not only the direct child, because a reviewer runs as
// `dotf secrets run -- pi …` and pi is a grandchild of this process.
func RunWithDeadline(timeout time.Duration, argv []string, stdin io.Reader, stdout, stderr io.Writer) (int, bool, error) {
	if len(argv) == 0 {
		return -1, false, errors.New("no command to run")
	}
	c := exec.Command(argv[0], argv[1:]...) //nolint:gosec // argv is the launcher's own reviewer command
	c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr
	startInOwnGroup(c)
	if err := c.Start(); err != nil {
		return -1, false, err
	}
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()

	sigs, stopListening := interrupts()
	defer stopListening()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-done:
		return exitStatus(err)
	case <-timer.C:
		stopGroup(c, deadlineGrace, done)
		return ExitDeadline, true, nil
	case sig := <-sigs:
		// The runner has a process group of its own, so a signal meant for
		// this run (Ctrl-C, `tmux kill-session`) no longer reaches it by
		// itself. Pass it on, or the reviewer would outlive the session that
		// ran it.
		stopGroup(c, deadlineGrace, done)
		if s, ok := sig.(syscall.Signal); ok {
			return 128 + int(s), false, nil
		}
		return 1, false, nil
	}
}

// interrupts is where RunWithDeadline hears that it was itself asked to stop;
// a seam so a test can deliver a signal without sending one to the test binary.
var interrupts = func() (<-chan os.Signal, func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, stopSignals...)
	return c, func() { signal.Stop(c) }
}

func exitStatus(err error) (int, bool, error) {
	if err == nil {
		return 0, false, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), false, nil
	}
	return -1, false, err
}
