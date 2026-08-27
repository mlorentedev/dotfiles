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
	t.Setenv("DOTF_BITACORA_REPO", "mlorentedev/dotfiles")

	stdout, _, err := execute(t, "spec", "init", "CLI-007-dot-spec-init", "--issue", "358")
	if err != nil {
		t.Fatalf("spec init --issue: %v", err)
	}
	if !strings.Contains(stdout, "Work-gate OK: mlorentedev/dotfiles#358 is open") {
		t.Errorf("missing work-gate confirmation:\n%s", stdout)
	}
	proposal := readFile(t, filepath.Join(root, "specs", "CLI-007-dot-spec-init", "proposal.md"))
	if !strings.Contains(proposal, `issue: "mlorentedev/dotfiles#358"`) {
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
	t.Setenv("DOTF_BITACORA_REPO", "mlorentedev/dotfiles")
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

// TestSpecInitBitacoraRepoFlagOverrides guards HARNESS-023: --bitacora-repo
// sets BOTH the gate repo and the frontmatter issue: prefix, so a cross-repo
// work-gate (e.g. a kubelab spec gated by a knowledge issue) resolves to the
// right repo instead of the current one.
func TestSpecInitBitacoraRepoFlagOverrides(t *testing.T) {
	root := makeRepo(t)
	pinClock(t)
	stubGhCmd(t, "OPEN\tCross-repo gate")

	_, _, err := execute(t, "spec", "init", "CLI-007-dot-spec-init",
		"--issue", "358", "--bitacora-repo", "mlorentedev/knowledge")
	if err != nil {
		t.Fatalf("spec init --bitacora-repo: %v", err)
	}
	proposal := readFile(t, filepath.Join(root, "specs", "CLI-007-dot-spec-init", "proposal.md"))
	if !strings.Contains(proposal, `issue: "mlorentedev/knowledge#358"`) {
		t.Errorf("frontmatter should record the overridden repo:\n%s", proposal)
	}
}

// TestSpecInitUnresolvableRepoFails guards that a gated init refuses to fabricate
// an issue prefix when the repo can't be resolved (no flag, no env, no origin) —
// it errors pointing at --bitacora-repo rather than scaffolding a bogus "#N".
// stubGhCmd points PATH at a dir with only `gh`, so the origin-slug fallback
// (which shells out to `git`) finds no git and yields an empty slug.
func TestSpecInitUnresolvableRepoFails(t *testing.T) {
	makeRepo(t)
	stubGhCmd(t, "OPEN\tWhatever")
	_, _, err := execute(t, "spec", "init", "CLI-007-dot-spec-init", "--issue", "358")
	if err == nil {
		t.Fatalf("expected error when the bitácora repo is unresolvable")
	}
	if !strings.Contains(err.Error(), "bitacora-repo") {
		t.Errorf("error should point at --bitacora-repo, got: %v", err)
	}
}

// TestSpecInit_UsesLocalCalendarDate guards CLI-044 / lesson 228: spec created:
// is a calendar date and must be formatted from the local clock, not UTC. An
// evening spec at 18:30 in MDT (-0600) is 00:30 the next day in UTC; stamping it
// in UTC dates the spec tomorrow.
func TestSpecInit_UsesLocalCalendarDate(t *testing.T) {
	root := makeRepo(t)
	denver := time.FixedZone("MDT", -6*60*60)
	evening := time.Date(2026, 8, 24, 19, 30, 0, 0, denver)
	if evening.UTC().Format("2006-01-02") == evening.Format("2006-01-02") {
		t.Fatal("fixture is not timezone-sensitive; it cannot catch the regression")
	}

	prev := now
	now = func() time.Time { return evening }
	t.Cleanup(func() { now = prev })

	_, _, err := execute(t, "spec", "init", "AI-030-demo", "--force-no-gate")
	if err != nil {
		t.Fatalf("spec init: %v", err)
	}

	proposal := readFile(t, filepath.Join(root, "specs", "AI-030-demo", "proposal.md"))
	if !strings.Contains(proposal, `created: "2026-08-24"`) {
		t.Errorf("expected local date 2026-08-24 in created field, got:\n%s", proposal)
	}
	if strings.Contains(proposal, `created: "2026-08-25"`) {
		t.Errorf("created field erroneously stamped in UTC (tomorrow):\n%s", proposal)
	}
}

// readFile reads the full contents of path as a string or fails the test.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
