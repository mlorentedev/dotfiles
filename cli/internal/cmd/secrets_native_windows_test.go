//go:build windows

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #3569 through the real runChild, the launch `dotf secrets run -- pi` makes: a
// .cmd shim is the first `fakeagent` on PATH and the tool manager holds a
// native executable of that name. The check passing is not enough; the launch
// must start that executable and not the shim PATH holds (#3440). The test
// binary is the native program, and TestSecretsNativeHelperProcess its entry
// point.
func TestRunChild_StartsTheNativeExecutableBehindABatchShim(t *testing.T) {
	dir := t.TempDir()
	ran := filepath.Join(dir, "shim-ran")
	if err := os.WriteFile(filepath.Join(dir, "fakeagent.cmd"), []byte("@echo off\r\necho x> \""+ran+"\"\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	prev := nativeBehind
	t.Cleanup(func() { nativeBehind = prev })
	asked := 0
	nativeBehind = func(string) string { asked++; return os.Args[0] }

	out := filepath.Join(dir, "args")
	prompt := "a & b %PATH% ^ | < > \"q\"\nsecond line"
	var stdout, stderr bytes.Buffer
	code, err := runChild(
		[]string{"fakeagent", "-test.run=^TestSecretsNativeHelperProcess$", "--", prompt},
		append(os.Environ(), "SECRETS_NATIVE_HELPER_OUT="+out),
		strings.NewReader(""), &stdout, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("runChild: exit %d, %v\n%s%s", code, err, stdout.String(), stderr.String())
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("the native child did not run: %v", err)
	}
	if string(got) != prompt {
		t.Errorf("the child received %q, want %q", got, prompt)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("the batch shim ran")
	}
	// One resolution serves the check and the launch: a second could answer
	// differently, and each is a `mise which`.
	if asked != 1 {
		t.Errorf("the tool manager was asked %d times, want 1", asked)
	}
}

// TestSecretsNativeHelperProcess is the child of the test above, not a test of
// its own: it does nothing unless the parent set SECRETS_NATIVE_HELPER_OUT.
func TestSecretsNativeHelperProcess(t *testing.T) {
	out := os.Getenv("SECRETS_NATIVE_HELPER_OUT")
	if out == "" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	if err := os.WriteFile(out, []byte(strings.Join(args, " ")), 0o600); err != nil {
		t.Fatal(err)
	}
}
