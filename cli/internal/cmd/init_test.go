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

// TestInitDryRunCmd asserts `dotf init <dir> --dry-run` reports actions without writing to disk.
func TestInitDryRunCmd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "target")
	stdout, stderr, err := execute(t, "init", dir, "--dry-run", "--skip-github", "--skip-vault")
	if err != nil {
		t.Fatalf("`dotf init --dry-run` errored: %v\n%s", err, stdout+stderr)
	}
	if !strings.Contains(stdout+stderr, "[DRY-RUN]") {
		t.Errorf("expected [DRY-RUN] in output, got:\n%s", stdout+stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
		t.Errorf("dry-run wrote AGENTS.md to %s", dir)
	}
}

// TestInitAgentsDryRunCmd asserts `dotf init agents --dry-run` reports intent without modifying AGENTS.md.
func TestInitAgentsDryRunCmd(t *testing.T) {
	dir := t.TempDir()
	stdout, stderr, err := execute(t, "init", "agents", "--repo", dir, "--dry-run")
	if err != nil {
		t.Fatalf("`dotf init agents --dry-run` errored: %v\n%s", err, stdout+stderr)
	}
	if !strings.Contains(stdout+stderr, "[DRY-RUN]") {
		t.Errorf("expected [DRY-RUN] in output, got:\n%s", stdout+stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
		t.Errorf("dry-run wrote AGENTS.md to %s", dir)
	}
}

// TestInitLane_DryRun asserts `dotf init lane --dry-run` reports intent and writes nothing, and a real run writes the contract.
func TestInitLane_DryRun(t *testing.T) {
	dir := t.TempDir()
	stdout, stderr, err := execute(t, "init", "lane", "--repo", dir, "--dry-run")
	if err != nil {
		t.Fatalf("`dotf init lane --dry-run` errored: %v\n%s", err, stdout+stderr)
	}
	if !strings.Contains(stdout+stderr, "[DRY-RUN]") {
		t.Errorf("expected [DRY-RUN] in output, got:\n%s", stdout+stderr)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("dry-run wrote %d entries to %s", len(entries), dir)
	}

	stdout, stderr, err = execute(t, "init", "lane", "--repo", dir)
	if err != nil {
		t.Fatalf("`dotf init lane` errored: %v\n%s", err, stdout+stderr)
	}
	for _, f := range []string{".dotf/lane.json", ".dotf/lane-prompt.md", "AGENTS.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("expected %s after `dotf init lane`: %v", f, err)
		}
	}
}

// TestInit_SkipFlagsReachTheLaneStep drives the real command so the flag-to-option
// wiring is covered, not only InitOptions: a mis-mapped --skip-lane or
// --skip-agents must change what lands on disk.
func TestInit_SkipFlagsReachTheLaneStep(t *testing.T) {
	cases := []struct {
		name       string
		flag       string
		wantLane   bool
		wantAgents bool
	}{
		{"no skip flag scaffolds both", "", true, true},
		{"--skip-lane keeps AGENTS.md", "--skip-lane", false, true},
		{"--skip-agents also skips the lane step", "--skip-agents", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			args := []string{"init", dir, "--skip-github", "--skip-vault"}
			if tc.flag != "" {
				args = append(args, tc.flag)
			}
			if stdout, stderr, err := execute(t, args...); err != nil {
				t.Fatalf("`dotf %s` errored: %v\n%s", strings.Join(args, " "), err, stdout+stderr)
			}
			if _, err := os.Stat(filepath.Join(dir, ".dotf", "lane.json")); (err == nil) != tc.wantLane {
				t.Errorf(".dotf/lane.json present=%v, want %v", err == nil, tc.wantLane)
			}
			if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); (err == nil) != tc.wantAgents {
				t.Errorf("AGENTS.md present=%v, want %v", err == nil, tc.wantAgents)
			}
		})
	}
}
