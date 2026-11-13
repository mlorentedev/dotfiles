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

// The refusal is for batch files only: a native executable still starts, with
// a prompt full of the same characters. The test binary is the native program.
func TestExec_StillStartsANativeExecutable(t *testing.T) {
	if err := Exec([]string{os.Args[0], "-test.run=^$", "a & b %PATH% ^ | < > \""}); err != nil {
		t.Fatalf("Exec of a native executable: %v", err)
	}
}
