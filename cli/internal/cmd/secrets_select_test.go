package cmd

import (
	"os"
	"testing"
)

// SEC-002 AC1, the finding the adversarial review raised as a Major on a636844.
//
// The two path tests prove runChildPTY attaches a pty and runChild does not.
// Neither proves WHICH is chosen, because both call their function directly and
// bypass the call site. That left the single line the whole fix rests on with no
// test: inverting it would silently restore the bug -- interactive children back
// on a pipe -- while every other test in this package still passed.
//
// This exercises the isTerminal seam in both directions, which is what AC1 asks
// for by name.
func TestWantsInteractiveChild_FollowsTheTerminalSeam(t *testing.T) {
	orig := isTerminal
	t.Cleanup(func() { isTerminal = orig })

	var askedFor []uintptr
	isTerminal = func(fd uintptr) bool {
		askedFor = append(askedFor, fd)
		return true
	}
	gotTrue := wantsInteractiveChild()

	isTerminal = func(uintptr) bool { return false }
	gotFalse := wantsInteractiveChild()

	// On a platform that can allocate a pty, the seam decides outright. On
	// Windows interactiveChildSupported() is false and pins the answer to false
	// regardless, which is itself the contract worth asserting.
	if supported := interactiveChildSupported(); supported {
		if !gotTrue {
			t.Error("a terminal parent did NOT select the pty path; interactive children are back on a pipe (SEC-002)")
		}
	} else if gotTrue {
		t.Error("the pty path was selected on a platform that cannot allocate one")
	}

	if gotFalse {
		t.Error("a non-terminal parent selected the pty path; CI, pipelines and `pi -p` must stay on the pipe")
	}

	// The fd consulted must be stdout: that is what an interactive child
	// inspects before deciding to draw. Asking about stdin or stderr would
	// answer a different question and be right by accident on a dev box.
	if interactiveChildSupported() {
		if len(askedFor) != 1 {
			t.Fatalf("expected exactly one isTerminal query, got %d", len(askedFor))
		}
		if askedFor[0] != os.Stdout.Fd() {
			t.Errorf("the branch consulted fd %d, not stdout (%d)", askedFor[0], os.Stdout.Fd())
		}
	}
}
