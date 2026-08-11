package spec

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateID(t *testing.T) {
	valid := []string{
		"AI-001-ollama-public",
		"SDD-012b-guard", // sub-id letter convention
		"CLI-007-dot-spec-init",
		"BUG-007", // ticket with no slug
		"2026-05-13-foo",
		"ADR028-004", // AREA carrying digits (the ADR028-* bitacora series)
		"ADR028-004-classify-stateful-service-placement",
		"ADR028-004b", // digits in AREA and a sub-id letter together
	}
	for _, id := range valid {
		if err := ValidateID(id); err != nil {
			t.Errorf("ValidateID(%q) = %v, want nil", id, err)
		}
	}

	invalid := []string{
		"",
		"lowercase-001",   // area must be upper
		"AI-001-Bad_Slug", // underscore + caps not allowed in slug
		"AI-001-CAPS",
		"foo",
		"2026-5-13-foo", // unpadded date
	}
	for _, id := range invalid {
		if err := ValidateID(id); err == nil {
			t.Errorf("ValidateID(%q) = nil, want error", id)
		}
	}
}

func TestRenderSubstitutesAndFixesIssueFrontmatter(t *testing.T) {
	files, err := Render("CLI-007-dot-spec-init", "2026-06-13", "mlorentedev/knowledge", 358, "Port init-spec")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	proposal := files["proposal.md"]
	wantContains := []string{
		`id: "CLI-007-dot-spec-init"`,
		`created: "2026-06-13"`,
		// HARNESS-023: records the issue's real owner/repo, not a hardcoded
		// "dotfiles" — and the full owner/name form, not the bare repo.
		`issue: "mlorentedev/knowledge#358"`,
		"# CLI-007-dot-spec-init",
		"<!-- from issue #358: Port init-spec -->",
	}
	for _, w := range wantContains {
		if !strings.Contains(proposal, w) {
			t.Errorf("proposal missing %q\n---\n%s", w, proposal)
		}
	}

	// No raw placeholders survive in any rendered file.
	for name, content := range files {
		for _, ph := range []string{"<feature-id>", "{TITLE}", "{{date:YYYY-MM-DD}}"} {
			if strings.Contains(content, ph) {
				t.Errorf("%s still contains placeholder %q", name, ph)
			}
		}
	}

	// The ## Why comment sits between the heading and the body, mirroring
	// init-spec.sh's awk injection.
	if !strings.Contains(proposal, "## Why\n\n<!-- from issue #358: Port init-spec -->\n") {
		t.Errorf("Why comment not injected in the expected position:\n%s", proposal)
	}
}

// TestRenderStampsCreatedInAllFiles guards the CodeRabbit-found bug (dotfiles#359):
// tasks.md and verification.md once hard-coded the template authoring date in
// created:, so scaffolded specs inherited it. All three templates must now carry
// the {{date}} placeholder and have it substituted with the generation date.
func TestRenderStampsCreatedInAllFiles(t *testing.T) {
	files, err := Render("BUG-007-x", "2026-06-13", "", 0, "")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, name := range []string{"proposal.md", "tasks.md", "verification.md"} {
		if !strings.Contains(files[name], `created: "2026-06-13"`) {
			t.Errorf("%s did not stamp created: with the generation date:\n%s", name, files[name])
		}
	}
}

func TestRenderWithoutIssueLeavesFrontmatterEmpty(t *testing.T) {
	files, err := Render("BUG-007", "2026-06-13", "", 0, "")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	proposal := files["proposal.md"]
	if !strings.Contains(proposal, `issue: ""`) {
		t.Errorf("expected empty issue frontmatter with no issue, got:\n%s", proposal)
	}
	if strings.Contains(proposal, "<!-- from issue") {
		t.Errorf("did not expect a Why provenance comment with no issue")
	}
}

func TestScaffoldWritesFilesAndGuardsClobber(t *testing.T) {
	root := t.TempDir()

	warn, err := Scaffold(root, "CLI-007-dot-spec-init", "2026-06-13", "mlorentedev/dotfiles", 358, "Port init-spec")
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	if warn != "" {
		t.Errorf("unexpected warning: %q", warn)
	}
	for _, name := range []string{"proposal.md", "tasks.md", "verification.md"} {
		p := filepath.Join(root, "specs", "CLI-007-dot-spec-init", name)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}

	// Re-scaffolding the same id must refuse to clobber.
	if _, err := Scaffold(root, "CLI-007-dot-spec-init", "2026-06-13", "mlorentedev/dotfiles", 358, "Port init-spec"); err == nil {
		t.Errorf("expected clobber guard error on second scaffold")
	}
}

func TestScaffoldWarnsOnArchivedID(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "specs", "archive", "AI-001-x"), 0o755); err != nil {
		t.Fatal(err)
	}
	warn, err := Scaffold(root, "AI-001-x", "2026-06-13", "", 0, "")
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	if !strings.Contains(warn, "specs/archive/") {
		t.Errorf("expected archive warning, got %q", warn)
	}
	// Still scaffolds despite the warning.
	if _, err := os.Stat(filepath.Join(root, "specs", "AI-001-x", "proposal.md")); err != nil {
		t.Errorf("expected scaffold to proceed despite archive warning: %v", err)
	}
}

// stubGh writes a fake `gh` executable emitting body on stdout (exit 0) or, if
// exitNonZero, msg on stderr (exit 1), and points PATH at it exclusively.
func stubGh(t *testing.T, stdout, stderr string, exitNonZero bool) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("gh stub uses a POSIX shell script")
	}
	dir := t.TempDir()
	script := "#!/bin/sh\n"
	if stdout != "" {
		script += "printf '%s' " + shellQuote(stdout) + "\n"
	}
	if stderr != "" {
		script += "printf '%s' " + shellQuote(stderr) + " >&2\n"
	}
	if exitNonZero {
		script += "exit 1\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func TestGateOpenIssueReturnsTitle(t *testing.T) {
	stubGh(t, "OPEN\tMy open issue", "", false)
	title, err := Gate(42, "")
	if err != nil {
		t.Fatalf("Gate: %v", err)
	}
	if title != "My open issue" {
		t.Errorf("title = %q, want %q", title, "My open issue")
	}
}

func TestGateClosedIssueFails(t *testing.T) {
	stubGh(t, "CLOSED\tDone already", "", false)
	if _, err := Gate(42, ""); err == nil {
		t.Errorf("expected error for a closed work-gate issue")
	} else if !strings.Contains(err.Error(), "not open") {
		t.Errorf("error should mention not-open, got: %v", err)
	}
}

func TestGateMissingIssueFails(t *testing.T) {
	stubGh(t, "", "gh: could not resolve issue", true)
	if _, err := Gate(99999, ""); err == nil {
		t.Errorf("expected error when gh fails to find the issue")
	}
}
