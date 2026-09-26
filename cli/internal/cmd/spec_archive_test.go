package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedSpec writes a minimal ARCHIVABLE specs/<id>/ under root: the proposal the
// caller cares about, plus the passing review.md the CLI-034 gate now requires.
// Tests that exercise the gate itself write their own review instead.
func seedSpec(t *testing.T, root, id, proposal string) {
	t.Helper()
	dir := filepath.Join(root, "specs", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proposal.md"), []byte(proposal), 0o644); err != nil {
		t.Fatal(err)
	}
	review := "---\nspec: \"" + id + "\"\nverdict: \"PASS\"\nreviewed_sha: \"0000000000000000000000000000000000000000\"\n---\nno blocking findings\n"
	if err := os.WriteFile(filepath.Join(dir, "review.md"), []byte(review), 0o644); err != nil {
		t.Fatal(err)
	}
	// Answered promotions, so these cases pass the promotion pre-flight
	// (HARNESS-160); internal/spec's promotion_test.go covers its refusals.
	promotions := "## Promotion candidates\n\n" +
		"- [x] Lesson for the repo's `docs/lessons/`? no: a fixture\n" +
		"- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a fixture\n" +
		"- [x] New pattern candidate for `00_meta/patterns/`? no: a fixture\n"
	if err := os.WriteFile(filepath.Join(dir, "verification.md"), []byte(promotions), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSpecArchiveListedInSpecHelp(t *testing.T) {
	stdout, stderr, err := execute(t, "spec", "--help")
	if err != nil {
		t.Fatalf("spec --help: %v", err)
	}
	if !strings.Contains(stdout+stderr, "archive") {
		t.Errorf("spec help should list the archive command:\n%s", stdout+stderr)
	}
}

func TestSpecArchiveHappyPath(t *testing.T) {
	root := makeRepo(t)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing # draft | implementing\n---\n# AI-001-x\n")

	stdout, _, err := execute(t, "spec", "archive", "AI-001-x")
	if err != nil {
		t.Fatalf("spec archive: %v", err)
	}
	if !strings.Contains(stdout, "Archived") {
		t.Errorf("missing archive notice:\n%s", stdout)
	}
	got := readFile(t, filepath.Join(root, "specs", "archive", "AI-001-x", "proposal.md"))
	if !strings.Contains(got, "status: archived # draft | implementing") {
		t.Errorf("status not rewritten (comment must survive):\n%s", got)
	}
}

func TestSpecArchiveBlocksOnDrafts(t *testing.T) {
	root := makeRepo(t)
	seedSpec(t, root, "AI-001-x", "---\nstatus: draft\n---\n<!-- [AGENT-SUGGESTION] reconsider -->\n")

	_, _, err := execute(t, "spec", "archive", "AI-001-x")
	if err == nil {
		t.Fatalf("expected drafts to block the archive")
	}
	if !strings.Contains(err.Error(), "AGENT-SUGGESTION") {
		t.Errorf("error should name the blocking tag, got: %v", err)
	}
	// Source untouched, no archive created.
	if _, statErr := os.Stat(filepath.Join(root, "specs", "archive", "AI-001-x")); statErr == nil {
		t.Errorf("archive must not be created when blocked")
	}

	// --force-with-drafts overrides, and since SDD-042 only with a recorded
	// reason: see TestArchiveBypassRecordedViaCLI.
	if _, _, err := execute(t, "spec", "archive", "AI-001-x", "--force-with-drafts", "--reason", "test"); err != nil {
		t.Errorf("force-with-drafts should archive: %v", err)
	}
}

// SDD-042 AC4, end to end: the flag without --reason is refused and moves
// nothing; with it, the archived proposal.md carries the review_bypass: record.
func TestArchiveBypassRecordedViaCLI(t *testing.T) {
	root := makeRepo(t)
	pinClock(t)
	seedSpec(t, root, "AI-001-x", "---\nstatus: draft\n---\n<!-- [AGENT-DRAFT] todo -->\n")

	if _, _, err := execute(t, "spec", "archive", "AI-001-x", "--force-with-drafts"); err == nil || !strings.Contains(err.Error(), "--reason") {
		t.Fatalf("a bypass without --reason must be refused, naming --reason: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "specs", "AI-001-x")); err != nil {
		t.Fatalf("a refused bypass must move nothing: %v", err)
	}

	if _, _, err := execute(t, "spec", "archive", "AI-001-x", "--force-with-drafts", "--reason", "abandoned scaffold"); err != nil {
		t.Fatalf("a reasoned bypass should archive: %v", err)
	}
	got := readFile(t, filepath.Join(root, "specs", "archive", "AI-001-x", "proposal.md"))
	if !strings.Contains(got, "review_bypass: \"force-with-drafts; overrode: 1 unresolved draft tag(s); reason: abandoned scaffold; date: ") {
		t.Errorf("archived proposal.md should carry the bypass record:\n%s", got)
	}
}

func TestSpecArchiveAbandonedAndPR(t *testing.T) {
	root := makeRepo(t)
	pinClock(t)
	seedSpec(t, root, "AI-001-x", "---\nstatus: draft\n---\n")

	if _, _, err := execute(t, "spec", "archive", "AI-001-x", "--abandoned", "--pr", "https://x/pr/1"); err != nil {
		t.Fatalf("spec archive --abandoned --pr: %v", err)
	}
	got := readFile(t, filepath.Join(root, "specs", "archive", "_abandoned", "AI-001-x", "proposal.md"))
	if !strings.Contains(got, "status: abandoned") {
		t.Errorf("status not abandoned:\n%s", got)
	}
	if !strings.Contains(got, "PR: https://x/pr/1") {
		t.Errorf("PR url not recorded:\n%s", got)
	}
}

func TestSpecArchiveMissingSpecFails(t *testing.T) {
	makeRepo(t)
	_, _, err := execute(t, "spec", "archive", "NOPE-1")
	if err == nil {
		t.Fatalf("expected error for a missing spec")
	}
}

// The command hands the archive the real vault resolver, so a promoted pattern
// is checked where it lives: in $VAULT_PATH, not in the repository.
func TestSpecArchiveChecksAPromotedPatternInTheVault(t *testing.T) {
	root := makeRepo(t)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	verification := "## Promotion candidates\n\n" +
		"- [x] Lesson for the repo's `docs/lessons/`? no: a fixture\n" +
		"- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a fixture\n" +
		"- [x] New pattern candidate for `00_meta/patterns/`? yes: 00_meta/patterns/pattern-promotion-test.md\n"
	if err := os.WriteFile(filepath.Join(root, "specs", "AI-001-x", "verification.md"), []byte(verification), 0o644); err != nil {
		t.Fatal(err)
	}
	vaultDir := t.TempDir()
	t.Setenv("VAULT_PATH", vaultDir)

	if _, _, err := execute(t, "spec", "archive", "AI-001-x"); err == nil ||
		!strings.Contains(err.Error(), "pattern-promotion-test.md does not exist in the vault") {
		t.Fatalf("a pattern missing from the vault should refuse the archive, got %v", err)
	}

	pattern := filepath.Join(vaultDir, "00_meta", "patterns", "pattern-promotion-test.md")
	if err := os.MkdirAll(filepath.Dir(pattern), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pattern, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := execute(t, "spec", "archive", "AI-001-x")
	if err != nil {
		t.Fatalf("with the pattern in the vault, the archive should pass: %v", err)
	}
	if !strings.Contains(stdout, "promotions: every candidate in verification.md is answered") {
		t.Errorf("the archive should report the promotion check:\n%s", stdout)
	}
}
