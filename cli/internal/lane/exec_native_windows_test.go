//go:build windows

package lane

import (
	"os"
	"path/filepath"
	"testing"
)

// No test starts what this machine's mise holds behind a batch shim: a test
// that wants a native target sets its own (lesson 610).
func init() { nativeBehind = func(string) string { return "" } }

// #3569 through the real Exec: a .cmd shim is the first `fakeagent` on PATH and
// the tool manager holds a native executable of that name. The executable
// starts with the prompt byte for byte, and the shim never runs. The test
// binary is the native program, and TestExecHelperProcess its entry point.
func TestExec_StartsTheNativeExecutableBehindABatchShim(t *testing.T) {
	dir := t.TempDir()
	ran := filepath.Join(dir, "shim-ran")
	if err := os.WriteFile(filepath.Join(dir, "fakeagent.cmd"), []byte("@echo off\r\necho x> \""+ran+"\"\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	prev := nativeBehind
	t.Cleanup(func() { nativeBehind = prev })
	nativeBehind = func(name string) string {
		if name != "fakeagent" {
			t.Errorf("asked for %q", name)
		}
		return os.Args[0]
	}

	out := filepath.Join(dir, "args")
	t.Setenv("LANE_EXEC_HELPER_OUT", out)
	prompt := "a & b %PATH% ^ | < > \"q\"\nsecond line"
	if err := Exec([]string{"fakeagent", "-test.run=^TestExecHelperProcess$", "--", prompt}); err != nil {
		t.Fatalf("Exec: %v", err)
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
}
