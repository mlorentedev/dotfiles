package cmd

import "errors"

// codedError carries a process exit status alongside a message, so a command
// can distinguish outcomes that are all "non-zero" to cobra.
//
// The alternative already in this tree — os.Exit inside RunE, as
// `dotf secrets run` does — costs a re-exec harness in the test file, because
// os.Exit kills the test runner. This stays in-process and therefore testable
// without one.
type codedError struct {
	code int
	err  error
}

func (e *codedError) Error() string { return e.err.Error() }
func (e *codedError) Unwrap() error { return e.err }

// withExitCode tags an error with the status the process should exit with.
func withExitCode(code int, err error) error { return &codedError{code: code, err: err} }

// ExitCode is what main() exits with. Anything untagged is 1, which is both the
// previous behaviour and the fail-closed direction: a composer reading an
// unrecognised non-zero code must treat it as "the task failed", never as "the
// pool was busy, try the next one".
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ce *codedError
	if errors.As(err, &ce) {
		return ce.code
	}
	return 1
}
