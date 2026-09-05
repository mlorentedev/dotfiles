//go:build !windows

package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// interactiveChildSupported reports whether this build can hand a child a pty.
// Split by build tag rather than by a runtime check because the pty import
// itself is POSIX-only: `GOOS=windows go vet ./...` has to stay clean, and a
// Windows build break is invisible to a Linux-only loop while failing the whole
// package (#1075).
func interactiveChildSupported() bool { return true }

// runChildPTY runs the child attached to a pseudo-terminal, streaming its
// output through out (the redacting writer) to this process's stdout.
//
// The pipe path in runChild stays exactly as it was and remains the default;
// this is taken only when `dotf secrets run` is itself attached to a terminal.
// The two differ in one visible way, decided deliberately: a pty carries a
// SINGLE stream, so the child's stdout and stderr arrive merged. That is what a
// terminal does anyway, so this path is closer to running the child unwrapped,
// not further from it.
//
// What is NOT done here: passing os.Stdout through to the child untouched.
// That would make the TUI perfect and drop the #1459 redaction guarantee at
// exactly the point where a session transcript is being captured, which is the
// case the guarantee exists for.
func runChildPTY(argv, environ []string, out io.Writer) (int, error) {
	// The environment-introspection guard is not weakened by the new path.
	if err := assertSafeChildCommand(argv); err != nil {
		return 1, err
	}

	c := exec.Command(argv[0], argv[1:]...) //nolint:gosec // argv is the user's own command
	c.Env = environ

	ptmx, err := pty.Start(c)
	if err != nil {
		return 1, fmt.Errorf("allocate a pty for %s: %w", argv[0], err)
	}
	defer func() { _ = ptmx.Close() }()

	// Raw mode only if stdin is itself a terminal. `cmd < file` with a terminal
	// stdout is legitimate, and MakeRaw on a non-terminal fd fails.
	var restore func()
	if term.IsTerminal(int(os.Stdin.Fd())) {
		state, rawErr := term.MakeRaw(int(os.Stdin.Fd()))
		if rawErr == nil {
			restore = func() { _ = term.Restore(int(os.Stdin.Fd()), state) }
		}
	}
	// Restored on the normal path AND from the signal handler below: leaving a
	// terminal in raw mode outlives this process and breaks the user's shell,
	// so it must not depend on reaching the end of the function.
	if restore != nil {
		defer restore()
	}

	// SIGWINCH keeps the child's idea of the window the same as the user's;
	// without it a TUI draws at the size the terminal had when it started.
	// The initial send sets the size, since pty.Start does not inherit it.
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	winch <- syscall.SIGWINCH
	go func() {
		for range winch {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()

	// SIGTERM still has to reach the child. SIGINT deliberately does not:
	// in raw mode the terminal no longer generates it, and the child receives
	// ^C through the pty's own line discipline -- which is what happens when
	// the child is run directly, and the behaviour a TUI expects.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case s := <-sigs:
				if restore != nil {
					restore()
				}
				_ = c.Process.Signal(s)
			case <-done:
				return
			}
		}
	}()
	defer func() {
		close(done)
		signal.Stop(sigs)
		signal.Stop(winch)
		close(winch)
	}()

	// Parent stdin -> child. Best-effort and unsynchronised: it ends when the
	// pty closes, and a blocked read on a terminal cannot be cancelled.
	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()

	// Child -> redacting writer. On Linux, reading a pty master after the child
	// exits yields EIO rather than EOF; that is the normal end of the stream
	// here, not a failure to report.
	_, copyErr := io.Copy(out, ptmx)
	if copyErr != nil && !errors.Is(copyErr, syscall.EIO) {
		// The child may still have a useful exit status, so this is not fatal
		// on its own -- Wait below decides.
		_ = copyErr
	}

	if waitErr := c.Wait(); waitErr != nil {
		var ee *exec.ExitError
		if errors.As(waitErr, &ee) {
			return ee.ExitCode(), nil
		}
		return 1, fmt.Errorf("run %s: %w", argv[0], waitErr)
	}
	return 0, nil
}
