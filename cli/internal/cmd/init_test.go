package cmd

import (
	"os"
	"path/filepath"
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

// TestInitAgentsCreatesAgentsMd exercises `dotf init agents --repo <dir>` end to
// end: it writes a self-contained AGENTS.md (no $VAULT_PATH) and is idempotent.
func TestInitAgentsCreatesAgentsMd(t *testing.T) {
	dir := t.TempDir()

	stdout, stderr, err := execute(t, "init", "agents", "--repo", dir)
	if err != nil {
		t.Fatalf("`dotf init agents` errored: %v\n%s", err, stdout+stderr)
	}
	got, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md not written: %v", err)
	}
	if !strings.Contains(string(got), "## Spec-Driven Development") {
		t.Errorf("AGENTS.md missing the SDD section:\n%s", got)
	}
	if strings.Contains(string(got), "$VAULT_PATH") {
		t.Errorf("AGENTS.md leaks $VAULT_PATH:\n%s", got)
	}

	// Re-run is a safe no-op and says so.
	stdout2, stderr2, err := execute(t, "init", "agents", "--repo", dir)
	if err != nil {
		t.Fatalf("re-run errored: %v\n%s", err, stdout2+stderr2)
	}
	if !strings.Contains(stdout2+stderr2, "already present") {
		t.Errorf("re-run should report the section already present, got:\n%s", stdout2+stderr2)
	}
}

// TestInitGithubSkipsGracefullyWithoutGh asserts `dotf init github` degrades to a
// [WARN] + exit 0 when gh is absent (ADR-022 C7), so the orchestrator never
// aborts. PATH is pointed at an empty dir to guarantee gh is not found.
func TestInitGithubSkipsGracefullyWithoutGh(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	stdout, stderr, err := execute(t, "init", "github", "--repo", "owner/name")
	if err != nil {
		t.Fatalf("`dotf init github` should not error when gh is absent: %v", err)
	}
	if !strings.Contains(stdout+stderr, "[WARN]") {
		t.Errorf("expected a [WARN] skip, got:\n%s", stdout+stderr)
	}
}
