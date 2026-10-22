// Package reproduce runs the check that decides whether a reviewer's claimed
// defect exists (TOOL-017 AC4-AC6).
//
// A reviewer's class is a hypothesis; this tests it. The owner fixed what
// counts as a reproduction on #1062 (option A): a command run against
// committed code whose non-zero exit is recorded. Anything else, a model's
// judgement included, is not a reproduction and goes to a human.
//
// Four states, because two failures look alike and must not be confused. A
// check that never ran as designed (could-not-run) is not evidence of a
// defect, although it exits non-zero. A probe that did not change the tree
// (did-not-land) is not a refutation, although the check then passes.
package reproduce

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// State is the verdict of one reproduction.
type State string

const (
	Reproduced      State = "reproduced"        // the check failed: the defect exists
	DidNotReproduce State = "did-not-reproduce" // the check passed: the claim is refuted
	DidNotLand      State = "did-not-land"      // the probe did not change the tree
	CouldNotRun     State = "could-not-run"     // the check never ran as designed
)

// ErrDirty refuses a tree with uncommitted changes.
var ErrDirty = errors.New("the tree has uncommitted changes: a reproduction runs against committed code")

// Options describe one reproduction.
type Options struct {
	Dir     string        // repository root
	Command []string      // the check; its exit decides the state
	Timeout time.Duration // zero means no limit beyond ctx
}

// outputTail bounds the output a record carries: the end of a failing
// check's output is where it says why.
const outputTail = 2048

// Result is one reproduction's record.
type Result struct {
	State   State
	Exit    int    // -1 when the command did not start or was killed
	Head    string // the commit the check ran on
	Command []string
	Output  string // the tail of the command's combined output
}

// Run runs the check once against the committed tree and classifies it.
// An error means no reproduction was attempted; every outcome of the check
// itself, a failure to start included, is a Result.
func Run(ctx context.Context, o Options) (Result, error) {
	if len(o.Command) == 0 {
		return Result{}, errors.New("no command: a reproduction is the exit of a check")
	}
	dirty, err := git(ctx, o.Dir, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return Result{}, err
	}
	if dirty != "" {
		return Result{}, fmt.Errorf("%w:\n%s", ErrDirty, dirty)
	}
	head, err := git(ctx, o.Dir, "rev-parse", "HEAD")
	if err != nil {
		return Result{}, err
	}

	if o.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, o.Command[0], o.Command[1:]...) //nolint:gosec // the check is the operator's own command
	cmd.Dir = o.Dir
	cmd.Stdout, cmd.Stderr = &out, &out
	started := cmd.Start() == nil
	exit := -1
	if started {
		_ = cmd.Wait() // the exit code below is the result; Wait's error restates it
		exit = cmd.ProcessState.ExitCode()
	}
	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
	return Result{
		State:   classify(exit, started, timedOut),
		Exit:    exit,
		Head:    head,
		Command: o.Command,
		Output:  tail(out.String(), outputTail),
	}, nil
}

// classify maps one run of the check to a state. Only a check that ran to
// its own exit can reproduce or refute: a start failure, a shell's 126 or
// 127, a signal (exit -1) or a timeout says nothing about the defect.
func classify(exit int, started, timedOut bool) State {
	switch {
	case !started, timedOut, exit < 0, exit == 126, exit == 127:
		return CouldNotRun
	case exit == 0:
		return DidNotReproduce
	default:
		return Reproduced
	}
}

func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, bytes.TrimSpace(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
