package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// waitDelay bounds how long we wait for a killed child's descriptors to close.
//
// Without it, exec's Wait blocks until every process holding the pipe exits —
// including grandchildren that inherited stdout. A harness that spawns a helper
// would therefore keep the dispatcher waiting long after the deadline killed
// the child, which would quietly undo the release-before-reap guarantee AC3
// makes. Small, because by this point we have already abandoned the work.
const waitDelay = 2 * time.Second

// runProcess is the one place this package forks anything. Both real backends
// ride it — argv is hive's transport, not a second mechanism — so kill-on-
// deadline, output capture and exit-code handling exist once rather than twice.
//
// stdin carries the task text where the tool accepts it: argv is world-readable
// through `ps` and is bounded by ARG_MAX, and a task is arbitrary user text.
func runProcess(ctx context.Context, bin string, args []string, stdin string) (stdout, stderr string, code int, err error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.WaitDelay = waitDelay
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var out, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errBuf

	runErr := cmd.Run()
	stdout, stderr = out.String(), errBuf.String()

	// stderr is returned whatever the exit status, and classification decides
	// what to make of it. Discarding it on a zero exit is what hid the measured
	// case: `pi --print` with no credential exits 0, answers nothing, and puts
	// the only explanation — `No models match pattern …` — on stderr.
	if runErr == nil {
		return stdout, stderr, 0, nil
	}
	var ee *exec.ExitError
	if errors.As(runErr, &ee) {
		return stdout, stderr, ee.ExitCode(), nil
	}
	// Could not run it at all — binary vanished between probe and dispatch,
	// permission denied, context cancelled before start.
	return stdout, stderr, -1, runErr
}

func combine(stdout, stderr string) string {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return stdout
	}
	if strings.TrimSpace(stdout) == "" {
		return stderr
	}
	return stdout + "\n" + stderr
}

// binaryPresent answers the `bin:NAME` probe: is this harness on PATH.
//
// It is the dispatch-time question, and it is deliberately NOT the map's
// declared pool probe. `pools.nan.probe` is `env:NAN_API_KEY`, and that
// variable is empty in any ordinary environment here because the key is a
// registered secret resolved just-in-time (ADR-028) and pi decrypts it itself.
// Reading the map's probe would therefore report nan unreachable on a machine
// where it is perfectly reachable. The map declares what a pool IS; this
// answers whether a transport can serve it right now.
func binaryPresent(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// classifyProcess turns a completed process into the seam's vocabulary.
//
// Fail-closed, and the direction is the contract's: a process that RAN and
// exited non-zero is a task failure, never an unavailable pool. Neither
// `claude -p` nor `pi --print` has an exit code meaning "rate limited", and
// grepping stderr for one would be the fragile proxy this repo keeps catching.
// Unavailability is answered before dispatch, by the probe, so it never has to
// be inferred from output afterwards.
//
// The empty-output rule is measured, not defensive. `printf … | pi --print` on
// a machine without the NaN credential exits **0**, prints `No models match
// pattern "nan/…"` on stderr and answers nothing. A dispatcher that reported
// that as `ok` would hand its caller a successful record with no answer in it,
// which is worse than a failure because nothing downstream would look twice.
func classifyProcess(stdout, stderr string, code int, runErr error) Response {
	if runErr != nil {
		return Response{Status: StatusTaskFailed, Exit: 1,
			Output: fmt.Sprintf("could not run the backend: %v", runErr)}
	}
	if code != 0 {
		// The child's stderr is the useful half of the report; a bare exit code
		// sends the reader back to a terminal to reproduce it.
		return Response{Status: StatusTaskFailed, Exit: code, Output: combine(stdout, stderr)}
	}
	if strings.TrimSpace(stdout) == "" {
		return Response{Status: StatusTaskFailed, Exit: 0, Output: combine(stderr,
			"the backend exited 0 and produced no output; a dispatch that answers nothing "+
				"has not done the task, whatever its exit code says")}
	}
	return Response{Status: StatusOK, Exit: 0, Output: stdout}
}
