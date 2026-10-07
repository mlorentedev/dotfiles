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
		// Hyphen-joined AREA segments (#1479): the live board convention, e.g.
		// kubelab's APP-CONFIG-*, CI-GATE-* and SEC-VIKUNJA-* series.
		"APP-CONFIG-015",
		"CI-GATE-018-flaky-guard",
		"SEC-VIKUNJA-001b",
		"APP-CONFIG2-003", // a segment may carry digits, like ADR028
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
		"2026-5-13-foo",  // unpadded date
		"APP--001",       // empty AREA segment
		"-APP-001",       // leading hyphen
		"APP-config-001", // every AREA segment is upper case
		"APP-CONFIG",     // no ticket number
		"APP-CONFIG-",    // hyphen with no number after it
		"APP-001-CONFIG", // an upper-case segment after the number is not a slug
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

	features := files["features.json"]
	if !strings.Contains(features, `"id": "CLI-007-dot-spec-init-f1"`) {
		t.Errorf("features.json missing substituted id:\n%s", features)
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
	for _, name := range []string{"proposal.md", "tasks.md", "verification.md", "features.json"} {
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

// shellQuote now lives in review_launch.go, where the launcher needs it for
// real: this file's copy was byte-identical, so the test uses the production
// one rather than keeping a second implementation of the same rule.

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

// #1452: the gate asks REST, whose quota GraphQL's exhaustion does not touch,
// and reads REST's lower-case state as open.
func TestGateAsksRESTForTheIssueInItsHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("gh stub uses a POSIX shell script")
	}
	for _, tc := range []struct{ repo, path string }{
		{"owner/name", "repos/owner/name/issues/42"},
		{"", "repos/{owner}/{repo}/issues/42"},
	} {
		dir := t.TempDir()
		argsFile := filepath.Join(dir, "args")
		script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + shellQuote(argsFile) + "\nprintf 'open\\tMy open issue'\n"
		if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", dir)
		title, err := Gate(42, tc.repo)
		if err != nil || title != "My open issue" {
			t.Fatalf("repo %q: Gate = (%q, %v), want the title: REST reports state in lower case", tc.repo, title, err)
		}
		got, err := os.ReadFile(argsFile)
		if err != nil {
			t.Fatal(err)
		}
		if args := strings.Split(strings.TrimSpace(string(got)), "\n"); len(args) < 2 || args[0] != "api" || args[1] != tc.path {
			t.Errorf("repo %q: gh called with %q, want api %s", tc.repo, args, tc.path)
		}
	}
}

func TestGateMissingIssueFails(t *testing.T) {
	stubGh(t, "", "gh: could not resolve issue", true)
	if _, err := Gate(99999, ""); err == nil {
		t.Errorf("expected error when gh fails to find the issue")
	}
}

// TestValidateIDErrorNamesTheRule pins #1479's second outcome: the refusal
// states the grammar instead of listing examples for the reader to generalise
// from. An examples-only message is how ADR028-004 read as proof that any
// AREA shape was fine, and how CI-GATE-* spread across 17 tickets before the
// first spec attempt discovered the rule.
func TestValidateIDErrorNamesTheRule(t *testing.T) {
	err := ValidateID("app-config-015")
	if err == nil {
		t.Fatal("ValidateID(\"app-config-015\") = nil, want error")
	}
	for _, want := range []string{"upper-case", "hyphen", "APP-CONFIG-015"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not name the rule (missing %q):\n%s", want, err)
		}
	}
}

func TestRecordWIPOverrideRefusesAProposalWithoutFrontmatter(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "specs", "BUG-001-x")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("# no frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RecordWIPOverride(root, "BUG-001-x", "r", 10, 10, "2026-09-30"); err == nil {
		t.Error("RecordWIPOverride accepted a proposal with no frontmatter")
	}
}
