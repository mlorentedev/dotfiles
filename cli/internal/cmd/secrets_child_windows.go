//go:build windows

package cmd

import (
	"errors"
	"io"
)

// interactiveChildSupported reports false on Windows, so `dotf secrets run`
// keeps the pipe path it has always used there.
//
// This is a deliberate scope boundary, not an oversight. A Windows pseudo-console
// (ConPTY) is a different API from a POSIX pty, needs its own resize and raw-mode
// handling, and cannot be exercised by the Linux CI leg that would have to prove
// it. Shipping an untested ConPTY path would be worse than the known limitation:
// SEC-002's defect is that an interactive child silently does nothing, and a
// half-working pseudo-console fails the same silent way.
//
// Consequence, stated rather than discovered: an interactive TUI launched through
// `dotf secrets run` on Windows still receives a pipe and still will not start.
// The workaround there remains what it is today -- run the tool directly.
func interactiveChildSupported() bool { return false }

// errNoPTY is returned if runChildPTY is ever reached on Windows. It should be
// unreachable: the caller guards on interactiveChildSupported() first. It exists
// so the guard failing is a loud error rather than a nil-pointer panic.
var errNoPTY = errors.New("a pseudo-terminal child is not supported on windows; the pipe path should have been selected")

func runChildPTY(_, _ []string, _ io.Writer) (int, error) {
	return 1, errNoPTY
}
