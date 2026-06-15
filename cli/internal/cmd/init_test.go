package cmd

import (
	"strings"
	"testing"
)

// TestInitCmdWired asserts `dotf init` is registered, runnable (so it renders the
// Usage block and is listed as a command, not a help topic), and that its help
// states the scaffolder's purpose and self-containment promise.
func TestInitCmdWired(t *testing.T) {
	stdout, stderr, err := execute(t, "init", "--help")
	if err != nil {
		t.Fatalf("`dotf init --help` errored: %v", err)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, "Usage:") {
		t.Errorf("`dotf init --help` missing the Usage block (is init runnable?)\n---\n%s", combined)
	}
	lower := strings.ToLower(combined)
	for _, want := range []string{"scaffold", "$vault_path"} {
		if !strings.Contains(lower, want) {
			t.Errorf("`dotf init --help` output missing %q\n---\n%s", want, combined)
		}
	}
}

// TestInitListedInRootHelp guards that init is a first-class command (listed under
// "Available Commands"), so a user discovers it via `dotf --help` — not relegated
// to "Additional help topics".
func TestInitListedInRootHelp(t *testing.T) {
	stdout, _, err := execute(t, "--help")
	if err != nil {
		t.Fatalf("`dotf --help` errored: %v", err)
	}
	avail := stdout
	if i := strings.Index(stdout, "Available Commands:"); i >= 0 {
		avail = stdout[i:]
		if j := strings.Index(avail, "\nFlags:"); j >= 0 {
			avail = avail[:j]
		}
	}
	if !strings.Contains(avail, "init") {
		t.Errorf("`dotf --help` does not list init under Available Commands:\n%s", stdout)
	}
}
