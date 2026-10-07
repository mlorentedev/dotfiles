// Package execerr keeps a failed command's own explanation in its error.
//
// exec.Cmd.Output captures stderr into *exec.ExitError, but ExitError.Error
// renders only "exit status N", so a caller that wraps it with %w reports a
// failure with the reason removed (#2087).
package execerr

import (
	"errors"
	"os/exec"
	"strings"
)

// WithStderr returns err with the stderr an *exec.ExitError captured appended
// to its message. It still unwraps to the original error, so errors.As finds
// the *exec.ExitError and its exit code. Any other error, or an exit with
// nothing on stderr, is returned unchanged.
func WithStderr(err error) error {
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return err
	}
	msg := strings.TrimSpace(string(ee.Stderr))
	if msg == "" {
		return err
	}
	return &stderrError{err: err, stderr: msg}
}

type stderrError struct {
	err    error
	stderr string
}

func (e *stderrError) Error() string { return e.err.Error() + ": " + e.stderr }
func (e *stderrError) Unwrap() error { return e.err }
