//go:build !windows

package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// SEC-002 AC1, the half that proves the fix. `test -t 1` is the child asking
// the exact question pi asks: is my stdout a terminal? On the pty path it must
// answer yes.
func TestRunChildPTY_ChildSeesATerminal(t *testing.T) {
	var buf bytes.Buffer
	code, err := runChildPTY([]string{"sh", "-c", "test -t 1"}, nil, &buf)
	if err != nil {
		t.Fatalf("runChildPTY: %v", err)
	}
	if code != 0 {
		t.Errorf("child reported stdout is NOT a terminal (exit %d); the pty was not attached", code)
	}
}

// SEC-002 AC1, the other half: the pipe path must keep behaving as it does
// today. This is also the regression test for the defect itself -- before the
// fix, this was the ONLY behaviour, which is why pi exited silently.
func TestRunChild_ChildSeesAPipe(t *testing.T) {
	var buf bytes.Buffer
	rw := newRedactWriter(&buf, nil)
	code, err := runChild([]string{"sh", "-c", "test -t 1"}, nil, strings.NewReader(""), rw, rw)
	if err != nil {
		t.Fatalf("runChild: %v", err)
	}
	if code == 0 {
		t.Error("the pipe path gave the child a terminal; the two paths are supposed to differ")
	}
}

// SEC-002 AC2. The redaction guarantee must survive the new path, including
// when the secret is split across two of the child's writes -- which is the
// normal case for a TUI, whose output is chunked and interleaved with escape
// sequences.
func TestRunChildPTY_RedactsASecretSplitAcrossWrites(t *testing.T) {
	const secret = "mock-openrouter-test-token-val"
	injected := []string{"OPENROUTER_API_KEY=" + secret}

	var buf bytes.Buffer
	rw := newRedactWriter(&buf, injected)

	// Two writes with a gap, so the secret genuinely crosses a boundary rather
	// than arriving in one buffer.
	script := `printf %s "mock-open"; sleep 0.2; printf "%s\n" "router-test-token-val"`
	code, err := runChildPTY([]string{"sh", "-c", script}, nil, rw)
	if err != nil {
		t.Fatalf("runChildPTY: %v", err)
	}
	if code != 0 {
		t.Fatalf("child exited %d", code)
	}
	if err := rw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, secret) {
		t.Error("the secret reached the parent's stdout in full through the pty")
	}
	if !strings.Contains(got, "[REDACTED:OPENROUTER_API_KEY]") {
		t.Errorf("the secret was not replaced by its placeholder; got %q", got)
	}
}

// The #1459 introspection guard is not weakened by the new path. Without this,
// adding the pty branch would have silently created a second way to run `env`
// under injected secrets.
func TestRunChildPTY_HonoursTheIntrospectionGuard(t *testing.T) {
	var buf bytes.Buffer
	_, err := runChildPTY([]string{"env"}, nil, &buf)
	if err == nil {
		t.Fatal("runChildPTY ran an introspection command; the #1459 guard does not cover the pty path")
	}
	if !strings.Contains(err.Error(), "introspection") {
		t.Errorf("failed for the wrong reason: %v", err)
	}
}
