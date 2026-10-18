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

// ExitStalled is the status of a run the stall detector stopped. It is not 124:
// the cap and a wedge are different causes, and the exit status is what a
// script reading the run sees.
const ExitStalled = 125

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

// Limits bound one reviewer run: the cap on the whole run, and the time its
// transcript may stay unchanged while the runner is alive.
type Limits struct {
	Deadline time.Duration
	// StallTimeout stops a run whose Transcript has not changed for this long,
	// before the cap does. Zero turns the detector off, and so does an empty
	// Transcript: there is nothing to watch.
	StallTimeout time.Duration
	Transcript   string
}

// StopReason says what ended a run, when the launcher did.
type StopReason int

const (
	StopNone     StopReason = iota // the runner ended on its own, or a signal did
	StopDeadline                   // the cap
	StopStall                      // a transcript that stopped changing
)

// RunResult is how a run ended. LastCall names the last tool call the
// transcript recorded when a stall stopped the run.
type RunResult struct {
	Code     int
	Stop     StopReason
	LastCall string
}

// RunWithDeadline runs argv under the cap alone. See RunWithLimits.
func RunWithDeadline(timeout time.Duration, argv []string, stdin io.Reader, stdout, stderr io.Writer) (int, bool, error) {
	res, err := RunWithLimits(Limits{Deadline: timeout}, argv, stdin, stdout, stderr)
	return res.Code, res.Stop == StopDeadline, err
}

// RunWithLimits runs argv and stops it, with every process it started, when the
// cap passes or its transcript stalls. It returns the runner's exit status, or
// ExitDeadline or ExitStalled when the launcher ended the run.
//
// This is how `dotf spec review --timeout` binds every runner (HARNESS-152).
// Before it, only agy was bounded, through its own --print-timeout; pi and any
// runner added later ran for as long as they ran. The whole process group is
// stopped, not only the direct child, because a reviewer runs as
// `dotf secrets run -- pi …` and pi is a grandchild of this process.
//
// The stall detector is CLI-094 (#1975): a tool call that never returns held a
// review for six minutes and, with only the cap, would have held it for 45.
func RunWithLimits(l Limits, argv []string, stdin io.Reader, stdout, stderr io.Writer) (RunResult, error) {
	if len(argv) == 0 {
		return RunResult{Code: -1}, errors.New("no command to run")
	}
	c := exec.Command(argv[0], argv[1:]...) //nolint:gosec // argv is the launcher's own reviewer command
	c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr
	startInOwnGroup(c)
	if err := c.Start(); err != nil {
		return RunResult{Code: -1}, err
	}
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()

	sigs, stopListening := interrupts()
	defer stopListening()
	timer := time.NewTimer(l.Deadline)
	defer timer.Stop()
	watch := newStallWatch(l)
	defer watch.stop()
	for {
		select {
		case err := <-done:
			code, _, err := exitStatus(err)
			return RunResult{Code: code}, err
		case <-timer.C:
			stopGroup(c, deadlineGrace, done)
			return RunResult{Code: ExitDeadline, Stop: StopDeadline}, nil
		case <-watch.tick:
			if watch.stalled() {
				last := LastToolCall(l.Transcript)
				stopGroup(c, deadlineGrace, done)
				return RunResult{Code: ExitStalled, Stop: StopStall, LastCall: last}, nil
			}
		case sig := <-sigs:
			// The runner has a process group of its own, so a signal meant for
			// this run (Ctrl-C, `tmux kill-session`) no longer reaches it by
			// itself. Pass it on, or the reviewer would outlive the session that
			// ran it.
			stopGroup(c, deadlineGrace, done)
			if s, ok := sig.(syscall.Signal); ok {
				return RunResult{Code: 128 + int(s)}, nil
			}
			return RunResult{Code: 1}, nil
		}
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
