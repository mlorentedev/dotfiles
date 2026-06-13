package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// makeRepo creates a temp directory that looks like a repo root (has .git) and
// chdir's into it for the duration of the test.
func makeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root
}

// pinClock fixes the scaffold clock so created: is deterministic.
func pinClock(t *testing.T) {
	t.Helper()
	prev := now
	now = func() time.Time { return time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = prev })
}

// stubGhCmd points PATH at a fake `gh` emitting stdout (exit 0).
func stubGhCmd(t *testing.T, stdout string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("gh stub uses a POSIX shell script")
	}
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s' '" + stdout + "'\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestSpecListedInRoot(t *testing.T) {
	stdout, stderr, err := execute(t, "--help")
	if err != nil {
		t.Fatalf("help: %v", err)
	}
	if !strings.Contains(stdout+stderr, "spec") {
		t.Errorf("root help should list the spec command:\n%s", stdout+stderr)
	}
}

func TestSpecInitForceNoGateScaffolds(t *testing.T) {
	root := makeRepo(t)
	pinClock(t)

	stdout, _, err := execute(t, "spec", "init", "BUG-007-demo", "--force-no-gate")
	if err != nil {
		t.Fatalf("spec init --force-no-gate: %v", err)
	}
	if !strings.Contains(stdout, "Created: specs/BUG-007-demo") {
		t.Errorf("missing creation notice:\n%s", stdout)
	}
	proposal := readFile(t, filepath.Join(root, "specs", "BUG-007-demo", "proposal.md"))
	if !strings.Contains(proposal, `created: "2026-06-13"`) {
		t.Errorf("created date not stamped from pinned clock:\n%s", proposal)
	}
	// No gate, no issue -> frontmatter stays empty (no fabricated issue).
	if !strings.Contains(proposal, `issue: ""`) {
		t.Errorf("expected empty issue frontmatter under --force-no-gate, got:\n%s", proposal)
	}
}

func TestSpecInitWithOpenIssueSetsFrontmatter(t *testing.T) {
	root := makeRepo(t)
	pinClock(t)
	stubGhCmd(t, "OPEN\tPort init-spec")

	stdout, _, err := execute(t, "spec", "init", "CLI-007-dot-spec-init", "--issue", "358")
	if err != nil {
		t.Fatalf("spec init --issue: %v", err)
	}
	if !strings.Contains(stdout, "Work-gate OK: issue #358 is open") {
		t.Errorf("missing work-gate confirmation:\n%s", stdout)
	}
	proposal := readFile(t, filepath.Join(root, "specs", "CLI-007-dot-spec-init", "proposal.md"))
	if !strings.Contains(proposal, `issue: "dotfiles#358"`) {
		t.Errorf("frontmatter issue not populated (the fix):\n%s", proposal)
	}
	if !strings.Contains(proposal, "<!-- from issue #358: Port init-spec -->") {
		t.Errorf("Why provenance comment not injected:\n%s", proposal)
	}
}

func TestSpecInitMissingGateFails(t *testing.T) {
	makeRepo(t)
	_, _, err := execute(t, "spec", "init", "CLI-007-dot-spec-init")
	if err == nil {
		t.Fatalf("expected error when no --issue and no --force-no-gate")
	}
	if !strings.Contains(err.Error(), "work-gate") {
		t.Errorf("error should name the work-gate, got: %v", err)
	}
}

func TestSpecInitClosedIssueFails(t *testing.T) {
	root := makeRepo(t)
	stubGhCmd(t, "CLOSED\tAlready done")
	_, _, err := execute(t, "spec", "init", "CLI-007-dot-spec-init", "--issue", "358")
	if err == nil {
		t.Fatalf("expected error for a closed work-gate issue")
	}
	// The gate fails before any directory is created.
	if _, statErr := os.Stat(filepath.Join(root, "specs", "CLI-007-dot-spec-init")); statErr == nil {
		t.Errorf("spec dir must not be created when the gate fails")
	}
}

func TestSpecInitInvalidIDFails(t *testing.T) {
	makeRepo(t)
	_, _, err := execute(t, "spec", "init", "bad_id", "--force-no-gate")
	if err == nil {
		t.Fatalf("expected error for an invalid feature-id")
	}
	if !strings.Contains(err.Error(), "invalid feature-id") {
		t.Errorf("error should name the invalid id, got: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
