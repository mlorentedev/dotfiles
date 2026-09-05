package cmd

import (
	"bytes"
	"os"
	"strings"
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

// A tail that is a proper prefix of a secret must not be written out raw when
// the stream ends -- those are the leading bytes of a credential, and Flush is
// exactly the moment the transcript is closed.
func TestRedactWriter_DoesNotFlushAPartialSecretPrefixRaw(t *testing.T) {
	const secret = "mock-openrouter-test-token-val"
	injected := []string{"OPENROUTER_API_KEY=" + secret}

	var buf bytes.Buffer
	rw := newRedactWriter(&buf, injected)

	// The child dies mid-key: a long proper prefix and then nothing.
	if _, err := rw.Write([]byte("key is mock-openrouter-test-toke")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := rw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, "mock-open") {
		t.Errorf("Flush emitted credential material raw: %q", got)
	}
	if !strings.Contains(got, "[REDACTED:OPENROUTER_API_KEY]") {
		t.Errorf("the partial prefix was not reported as redacted material: %q", got)
	}
}

// If one injected value is a prefix of another, replacing the shorter first
// breaks the longer one's match and leaks its suffix. Order is free to fix.
func TestRedactWriter_HandlesOneSecretBeingAPrefixOfAnother(t *testing.T) {
	injected := []string{
		"SHORTKEY=mock-prefix-value",
		"LONGKEY=mock-prefix-value-with-more",
	}

	var buf bytes.Buffer
	rw := newRedactWriter(&buf, injected)

	if _, err := rw.Write([]byte("using mock-prefix-value-with-more now\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := rw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, "-with-more") {
		t.Errorf("the longer secret's suffix leaked because the shorter one matched first: %q", got)
	}
	if !strings.Contains(got, "[REDACTED:LONGKEY]") {
		t.Errorf("the longer secret was not redacted under its own name: %q", got)
	}
}
