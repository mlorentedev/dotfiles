package cmd

import (
	"bytes"
	"io"
	"os"
	"strconv"
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

func TestParseOnly(t *testing.T) {
	if parseOnly("") != nil || parseOnly("  ") != nil {
		t.Error("empty --only must be nil (= all)")
	}
	set := parseOnly("A, B ,,C")
	if len(set) != 3 || !set["A"] || !set["B"] || !set["C"] {
		t.Errorf("parseOnly = %v, want {A,B,C}", set)
	}
}
