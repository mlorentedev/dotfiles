package cmd

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestHelperProcess is not a real test: when re-exec'd with GO_WANT_HELPER_PROCESS=1
// it plays the role of `run`'s child — printing the injected FOO value and exiting
// with EXIT_CODE — so runChild's env-injection and exit-code propagation are
// exercised cross-platform without depending on a system shell.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	_, _ = io.WriteString(os.Stdout, os.Getenv("FOO"))
	code, _ := strconv.Atoi(os.Getenv("EXIT_CODE"))
	os.Exit(code)
}

func helperArgv() []string { return []string{os.Args[0], "-test.run=TestHelperProcess"} }

func TestRunChild_PropagatesExitCodeAndInjectsEnv(t *testing.T) {
	var out bytes.Buffer
	environ := append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "FOO=secret-value", "EXIT_CODE=3")
	code, err := runChild(helperArgv(), environ, nil, &out, io.Discard)
	if err != nil {
		t.Fatalf("runChild: %v", err)
	}
	if code != 3 {
		t.Errorf("exit code = %d, want 3 (child status propagated)", code)
	}
	if out.String() != "secret-value" {
		t.Errorf("child stdout = %q, want the injected FOO value", out.String())
	}
}

func TestRunChild_ZeroExit(t *testing.T) {
	environ := append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "FOO=x", "EXIT_CODE=0")
	code, err := runChild(helperArgv(), environ, nil, io.Discard, io.Discard)
	if err != nil || code != 0 {
		t.Fatalf("want (0,nil), got (%d,%v)", code, err)
	}
}

func TestRunChild_LaunchFailureIsError(t *testing.T) {
	if _, err := runChild([]string{"definitely-no-such-binary-xyz123"}, os.Environ(), nil, io.Discard, io.Discard); err == nil {
		t.Fatal("expected a launch error for a missing binary")
	}
}

func TestAssertSafeChildCommand(t *testing.T) {
	cases := []struct {
		name    string
		argv    []string
		wantErr bool
	}{
		{"empty argv", []string{}, true},
		{"bare env", []string{"env"}, true},
		{"path to env", []string{"/usr/bin/env"}, true},
		{"bare printenv", []string{"printenv"}, true},
		{"bare export", []string{"export"}, true},
		{"shell -c env", []string{"sh", "-c", "env | grep SECRET"}, true},
		{"quoted env in sh -c", []string{"sh", "-c", "'env'"}, true},
		{"double quoted env in sh -c", []string{"sh", "-c", "\"env\""}, true},
		{"escaped env in sh -c", []string{"sh", "-c", "\\env"}, true},
		{"bundled flag bash -lc", []string{"bash", "-lc", "env"}, true},
		{"interleaved flag bash -i -c", []string{"bash", "-i", "-c", "env"}, true},
		{"long flag bash --norc -c", []string{"bash", "--norc", "-c", "env"}, true},
		{"bash -c set", []string{"bash", "-c", "set"}, true},
		{"bash -c declare -p", []string{"bash", "-c", "declare -p"}, true},
		{"bash -c printenv", []string{"bash", "-c", "printenv FOO"}, true},
		{"bash -c export", []string{"bash", "-c", "export -p"}, true},
		{"allowed tool", []string{"goreleaser", "release"}, false},
		{"allowed python", []string{"python3", "script.py"}, false},
		{"allowed dotf review", []string{"dotf", "review"}, false},
		{"allowed echo env word", []string{"echo", "running in safe environment"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := assertSafeChildCommand(tc.argv)
			if (err != nil) != tc.wantErr {
				t.Errorf("assertSafeChildCommand(%v) err = %v, wantErr = %v", tc.argv, err, tc.wantErr)
			}
		})
	}
}

func TestRedactWriter_RedactsInjectedSecrets(t *testing.T) {
	injected := []string{
		"OPENROUTER_API_KEY=mock-openrouter-test-token-val",
		"NAN_API_KEY=mock-nan-test-token-val",
		"SHORT=abc", // len < 6, not redacted
	}

	var buf bytes.Buffer
	rw := newRedactWriter(&buf, injected)

	input := "Connecting with OPENROUTER_API_KEY=mock-openrouter-test-token-val and NAN_API_KEY=mock-nan-test-token-val, short is abc.\n"
	n, err := rw.Write([]byte(input))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(input) {
		t.Errorf("Write returned n = %d, want %d", n, len(input))
	}
	if err := rw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	got := buf.String()
	want := "Connecting with OPENROUTER_API_KEY=[REDACTED:OPENROUTER_API_KEY] and NAN_API_KEY=[REDACTED:NAN_API_KEY], short is abc.\n"
	if got != want {
		t.Errorf("redacted output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// SEC-002. The hold-back must be PREFIX-AWARE, not a fixed window.
//
// The rule this replaced withheld maxSecretLen-1 bytes on every write no matter
// what they were, so with a 49-byte token the last 48 bytes of every frame
// stayed invisible until more output arrived. Against a pipe drained at Flush()
// that is unobservable; against a terminal it is the whole defect -- the tail of
// a TUI frame is its cursor positioning.
func TestRedactWriter_ReleasesFrameWithNoSecretPrefixImmediately(t *testing.T) {
	injected := []string{"OPENROUTER_API_KEY=mock-openrouter-test-token-val"}

	var buf bytes.Buffer
	rw := newRedactWriter(&buf, injected)

	// Ordinary terminal output. Nothing here is a prefix of the secret, so all
	// of it must reach the target on this Write -- WITHOUT a Flush.
	frame := "\x1b[2J\x1b[H hello from the tui \x1b[10;20H"
	if _, err := rw.Write([]byte(frame)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if got := buf.String(); got != frame {
		t.Errorf("frame was not released on write (this is the SEC-002 defect):\ngot:  %q\nwant: %q", got, frame)
	}
}

// The complement: a trailing run that IS a proper prefix of a secret must still
// be withheld, or the redaction is defeated by chunking alone.
func TestRedactWriter_HoldsBackATrailingSecretPrefix(t *testing.T) {
	const secret = "mock-openrouter-test-token-val"
	injected := []string{"OPENROUTER_API_KEY=" + secret}

	var buf bytes.Buffer
	rw := newRedactWriter(&buf, injected)

	// Ends mid-secret: "mock-open" is a proper prefix and must not be emitted.
	if _, err := rw.Write([]byte("key is mock-open")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := buf.String(); got != "key is " {
		t.Errorf("the secret prefix leaked or too much was held:\ngot:  %q\nwant: %q", got, "key is ")
	}

	// Completing it must redact the whole value, never emit it.
	if _, err := rw.Write([]byte("router-test-token-val\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := rw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, secret) {
		t.Error("the secret reached the target in full after being split across two writes")
	}
	if want := "key is [REDACTED:OPENROUTER_API_KEY]\n"; got != want {
		t.Errorf("split-write redaction mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}
