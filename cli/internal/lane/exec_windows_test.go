//go:build windows

package lane

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An npm-installed agent is a .cmd shim, and cmd.exe would run what a free-text
// prompt smuggles in after `&` (#3429). The shim here writes a marker file when
// it runs at all, and the prompt carries a second command that writes another:
// Exec must start neither, whatever the prompt holds.
func TestExec_RefusesABatchShimAndNeverStartsIt(t *testing.T) {
	dir := t.TempDir()
	ran := filepath.Join(dir, "shim-ran")
	injected := filepath.Join(dir, "injected")
	for _, ext := range []string{".cmd", ".bat", ".CMD"} {
		name := "fakeagent" + strings.TrimPrefix(ext, ".")
		if err := os.WriteFile(filepath.Join(dir, name+ext), []byte("@echo off\r\necho x> \""+ran+"\"\r\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
		for _, prompt := range []string{
			"plain text",
			"fix it & echo x> \"" + injected + "\"",
			"100%PATH%",
			"a ^ b | c < d > e",
			`say "hi" & echo x> "` + injected + `"`,
			"two\nlines",
		} {
			err := Exec([]string{name, prompt})
			if err == nil || !strings.Contains(err.Error(), "cmd.exe") || !strings.Contains(err.Error(), name) {
				t.Errorf("%s, prompt %q: want a refusal naming the agent and cmd.exe, got %v", name+ext, prompt, err)
			}
		}
	}
	for _, f := range []string{ran, injected} {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("%s exists: the shim ran", filepath.Base(f))
		}
	}
}

// The refusal is for batch files only: a native executable still starts and
// receives the prompt byte for byte, with the same characters in it. The test
// binary is the native program, and TestExecHelperProcess its entry point.
func TestExec_StillStartsANativeExecutableWithThePromptIntact(t *testing.T) {
	out := filepath.Join(t.TempDir(), "args")
	t.Setenv("LANE_EXEC_HELPER_OUT", out)
	prompt := "a & b %PATH% ^ | < > \"q\"\nsecond line"
	if err := Exec([]string{os.Args[0], "-test.run=^TestExecHelperProcess$", "--", prompt}); err != nil {
		t.Fatalf("Exec of a native executable: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("the native child did not run: %v", err)
	}
	if string(got) != prompt {
		t.Errorf("the child received %q, want %q", got, prompt)
	}
}

// TestExecHelperProcess is the child of the test above, not a test of its own:
// it does nothing unless the parent set LANE_EXEC_HELPER_OUT.
func TestExecHelperProcess(t *testing.T) {
	out := os.Getenv("LANE_EXEC_HELPER_OUT")
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
