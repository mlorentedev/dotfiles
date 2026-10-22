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
	"maps"
	"os"
	"os/exec"
	"path/filepath"
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

// ErrNotReverted is a probe that could not be taken back out: the tree no
// longer matches the commit the record names.
var ErrNotReverted = errors.New("the probe could not be reverted")

// Options describe one reproduction.
type Options struct {
	Dir     string        // repository root
	Command []string      // the check; its exit decides the state
	Timeout time.Duration // zero means no limit beyond ctx
	Probe   string        // a patch applied for the run; its files are restored after it
}

// outputTail bounds the output a record carries: the end of a failing
// check's output is where it says why.
const outputTail = 2048

// waitDelay bounds how long the output of a killed check is still read.
const waitDelay = 5 * time.Second

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
	// A patch names files from the root, so the probe is applied and its
	// targets read there; the check alone runs in o.Dir.
	top, err := git(ctx, o.Dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return Result{}, err
	}

	res := Result{Exit: -1, Head: head, Command: o.Command}
	if o.Probe == "" {
		res.Exit, res.State, res.Output = check(ctx, o)
		return res, nil
	}
	return probed(ctx, o, top, res)
}

// probed applies the probe, runs the check only if the probe changed a
// file it names, and restores the targets whatever happened. Landing is decided by
// the targets' bytes, not by git apply's exit: an apply that succeeds and
// changes no content (a mode-only patch) did not land either.
func probed(ctx context.Context, o Options, top string, res Result) (Result, error) {
	if _, err := os.Stat(o.Probe); err != nil {
		return res, fmt.Errorf("probe: %w", err)
	}
	targets, err := probeTargets(ctx, top, o.Probe)
	if err != nil {
		res.State, res.Output = DidNotLand, err.Error()
		return res, nil
	}
	before := fingerprint(ctx, top, targets)
	if _, err := git(ctx, top, "apply", o.Probe); err != nil {
		res.State, res.Output = DidNotLand, err.Error()
		return res, nil
	}
	if maps.Equal(before, fingerprint(ctx, top, targets)) {
		res.State = DidNotLand
	} else {
		res.Exit, res.State, res.Output = check(ctx, o)
	}
	// The revert must not inherit a check's timeout: a probe left applied
	// is worse than any verdict.
	revertErr := restore(context.WithoutCancel(ctx), top, targets)
	if revertErr != nil || !maps.Equal(before, fingerprint(context.WithoutCancel(ctx), top, targets)) {
		return res, fmt.Errorf("%w: %s no longer match the commit %s (%v)", ErrNotReverted, strings.Join(targets, ", "), res.Head, revertErr)
	}
	return res, nil
}

// restore puts the targets back as HEAD has them, which is how they were
// found: the tree was clean. Unlike a reverse patch it does not depend on
// the check having left the probed lines alone. A target HEAD lacks was
// created by the probe and is removed.
func restore(ctx context.Context, top string, targets []string) error {
	tracked, err := git(ctx, top, append([]string{"ls-tree", "--name-only", "-z", "HEAD", "--"}, targets...)...)
	if err != nil {
		return err
	}
	inHead := map[string]bool{}
	var checkout []string
	for _, t := range strings.Split(tracked, "\x00") {
		if t != "" {
			inHead[t] = true
			checkout = append(checkout, t)
		}
	}
	if len(checkout) > 0 {
		if _, err := git(ctx, top, append([]string{"checkout", "HEAD", "--"}, checkout...)...); err != nil {
			return err
		}
	}
	for _, t := range targets {
		if inHead[t] {
			continue
		}
		if err := os.Remove(filepath.Join(top, t)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// probeTargets lists the files a patch touches, both sides of a rename.
// An empty or malformed patch is an error here, which the caller records
// as did-not-land; a patch naming no file changes no bytes, and lands
// nowhere by the same check.
func probeTargets(ctx context.Context, dir, probe string) ([]string, error) {
	out, err := git(ctx, dir, "apply", "--numstat", "-z", probe)
	if err != nil {
		return nil, err
	}
	var targets []string
	for _, field := range strings.Split(out, "\x00") {
		if i := strings.LastIndexByte(field, '\t'); i >= 0 {
			field = field[i+1:]
		}
		if field != "" {
			targets = append(targets, field)
		}
	}
	return targets, nil
}

// fingerprint is each target's content as git sees it: the blob id after
// the checkout's eol conversion and clean filters are undone. Raw bytes are
// the wrong frame: under autocrlf a file git calls clean may be LF on disk
// while every git write renders it CRLF, so the bytes change on apply and on
// restore although the content does not (measured on the Windows runner). A
// missing file is its own value, so a probe that creates or deletes a file
// lands too.
func fingerprint(ctx context.Context, top string, targets []string) map[string]string {
	sums := make(map[string]string, len(targets))
	for _, t := range targets {
		if _, err := os.Stat(filepath.Join(top, t)); err != nil {
			sums[t] = "absent"
			continue
		}
		id, err := git(ctx, top, "hash-object", "--", t)
		if err != nil {
			id = "unreadable: " + err.Error()
		}
		sums[t] = id
	}
	return sums
}

// check runs the command once and classifies its exit.
func check(ctx context.Context, o Options) (int, State, string) {
	if o.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, o.Command[0], o.Command[1:]...) //nolint:gosec // the check is the operator's own command
	cmd.Dir = o.Dir
	cmd.Stdout, cmd.Stderr = &out, &out
	// Killing the check does not close a pipe a child of it inherited;
	// without a bound, Wait would outlive the timeout by the child's life.
	cmd.WaitDelay = waitDelay
	started := cmd.Start() == nil
	exit := -1
	if started {
		_ = cmd.Wait() // the exit code below is the result; Wait's error restates it
		exit = cmd.ProcessState.ExitCode()
	}
	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
	return exit, classify(exit, started, timedOut), tail(out.String(), outputTail)
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
