package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedSpec writes a minimal specs/<id>/proposal.md under root for archive tests.
func seedSpec(t *testing.T, root, id, proposal string) {
	t.Helper()
	dir := filepath.Join(root, "specs", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proposal.md"), []byte(proposal), 0o644); err != nil {
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

	// --force-with-drafts overrides.
	if _, _, err := execute(t, "spec", "archive", "AI-001-x", "--force-with-drafts"); err != nil {
		t.Errorf("force-with-drafts should archive: %v", err)
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
