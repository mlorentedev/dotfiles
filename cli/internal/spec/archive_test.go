package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSpec materializes specs/<id>/ under root with the given files.
func writeSpec(t *testing.T, root, id string, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(root, "specs", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestFindUnresolvedTags(t *testing.T) {
	root := t.TempDir()
	dir := writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md":     "line one\n<!-- [AGENT-DRAFT] write the why -->\nline three\n",
		"tasks.md":        "- [ ] do it [AGENT-SUGGESTION] consider Y\n",
		"verification.md": "all clean here\n",
	})

	tags, err := FindUnresolvedTags(dir)
	if err != nil {
		t.Fatalf("FindUnresolvedTags: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("want 2 tagged lines, got %d: %v", len(tags), tags)
	}
	joined := strings.Join(tags, "\n")
	for _, want := range []string{"proposal.md", "AGENT-DRAFT", "tasks.md", "AGENT-SUGGESTION"} {
		if !strings.Contains(joined, want) {
			t.Errorf("tags output missing %q:\n%s", want, joined)
		}
	}
}

func TestFindUnresolvedTagsCleanDir(t *testing.T) {
	root := t.TempDir()
	dir := writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: implementing\n---\nno tags here\n",
	})
	tags, err := FindUnresolvedTags(dir)
	if err != nil {
		t.Fatalf("FindUnresolvedTags: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("want no tags, got %v", tags)
	}
}

func TestSetStatus(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		newStatus string
		want      string // substring that must be present after rewrite
		notWant   string // substring that must be absent after rewrite
	}{
		{
			name:      "frontmatter value replaced, enum comment kept",
			in:        "---\nstatus: implementing # draft | implementing | verifying | archived\n---\nbody\n",
			newStatus: "archived",
			want:      "status: archived # draft | implementing | verifying | archived",
		},
		{
			name:      "meaningful trailing comment preserved (DX-002 case)",
			in:        "---\nstatus: implementing # superseded by ADR-020\n---\nbody\n",
			newStatus: "abandoned",
			want:      "status: abandoned # superseded by ADR-020",
		},
		{
			name:      "no trailing comment",
			in:        "---\nstatus: draft\n---\nbody\n",
			newStatus: "archived",
			want:      "---\nstatus: archived\n---",
		},
		{
			name:      "decoy status line in body left untouched",
			in:        "---\nstatus: draft\n---\n\n## Notes\nstatus: pending stays as-is.\n",
			newStatus: "archived",
			want:      "status: pending stays as-is.",
			notWant:   "status: archived stays as-is.",
		},
		{
			name:      "no frontmatter => unchanged",
			in:        "no frontmatter here\nstatus: draft\n",
			newStatus: "archived",
			want:      "status: draft",
			notWant:   "status: archived",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := setStatus(tt.in, tt.newStatus)
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Errorf("setStatus missing %q:\n%s", tt.want, got)
			}
			if tt.notWant != "" && strings.Contains(got, tt.notWant) {
				t.Errorf("setStatus should not contain %q:\n%s", tt.notWant, got)
			}
		})
	}
}

// setStatus must rewrite only the first frontmatter block even when the body
// later opens a fenced YAML block that also has a status: line.
func TestSetStatusOnlyFirstBlock(t *testing.T) {
	in := "---\nstatus: draft\n---\n\nExample frontmatter in docs:\n\n---\nstatus: draft\n---\n"
	got := setStatus(in, "archived")
	if strings.Count(got, "status: archived") != 1 {
		t.Errorf("expected exactly one rewrite (first block only), got:\n%s", got)
	}
}

func TestArchiveMovesAndSetsStatus(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: implementing # draft | implementing\n---\n# AI-001-x\n",
		"tasks.md":    "tasks\n",
	})

	target, err := Archive(root, "AI-001-x", ArchiveOptions{})
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	want := filepath.Join(root, "specs", "archive", "AI-001-x")
	if target != want {
		t.Errorf("target = %q, want %q", target, want)
	}
	if _, err := os.Stat(filepath.Join(root, "specs", "AI-001-x")); !os.IsNotExist(err) {
		t.Errorf("source should be moved away, stat err = %v", err)
	}
	got := mustRead(t, filepath.Join(want, "proposal.md"))
	if !strings.Contains(got, "status: archived # draft | implementing") {
		t.Errorf("status not rewritten (comment must survive):\n%s", got)
	}
}

func TestArchiveAbandonedRoute(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{"proposal.md": "---\nstatus: draft\n---\n"})

	target, err := Archive(root, "AI-001-x", ArchiveOptions{Abandoned: true})
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	want := filepath.Join(root, "specs", "archive", "_abandoned", "AI-001-x")
	if target != want {
		t.Errorf("abandoned target = %q, want %q", target, want)
	}
	got := mustRead(t, filepath.Join(target, "proposal.md"))
	if !strings.Contains(got, "status: abandoned") {
		t.Errorf("status not abandoned:\n%s", got)
	}
}

func TestArchiveMissingSpecFails(t *testing.T) {
	root := t.TempDir()
	if _, err := Archive(root, "NOPE-1", ArchiveOptions{}); err == nil {
		t.Errorf("expected error for a missing spec")
	}
}

func TestArchiveNoClobber(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{"proposal.md": "---\nstatus: draft\n---\n"})
	if err := os.MkdirAll(filepath.Join(root, "specs", "archive", "AI-001-x"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{}); err == nil {
		t.Errorf("expected a no-clobber error")
	}
	if _, err := os.Stat(filepath.Join(root, "specs", "AI-001-x")); err != nil {
		t.Errorf("source must be left in place on no-clobber: %v", err)
	}
}

func TestArchiveBlocksOnDrafts(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: draft\n---\n<!-- [AGENT-DRAFT] todo -->\n",
	})

	_, err := Archive(root, "AI-001-x", ArchiveOptions{})
	if err == nil {
		t.Fatalf("expected drafts to block the archive")
	}
	if !strings.Contains(err.Error(), "AGENT-DRAFT") {
		t.Errorf("error should list the blocking tag, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "specs", "AI-001-x")); err != nil {
		t.Errorf("source must remain when blocked: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "specs", "archive", "AI-001-x")); !os.IsNotExist(err) {
		t.Errorf("target must not be created when blocked")
	}
}

func TestArchiveForceWithDrafts(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: draft\n---\n<!-- [AGENT-DRAFT] todo -->\n",
	})

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{ForceWithDrafts: true}); err != nil {
		t.Fatalf("force-with-drafts should archive despite tags: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "specs", "archive", "AI-001-x", "proposal.md")); err != nil {
		t.Errorf("expected archived proposal: %v", err)
	}
}

// TestArchiveRejectsTraversalID guards the path-traversal class CodeRabbit
// flagged on #362: a crafted id must be rejected by ValidateID before it can
// reach the filepath.Join calls and move something outside specs/.
func TestArchiveRejectsTraversalID(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"../outside", "../../etc", "foo/bar", ".", ".."} {
		if _, err := Archive(root, id, ArchiveOptions{ForceWithDrafts: true}); err == nil {
			t.Errorf("Archive(%q) = nil error, want rejection", id)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("a rejected traversal id must move nothing: %v", err)
	}
}

func TestArchiveRecordsPRURL(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{"proposal.md": "---\nstatus: draft\n---\nbody\n"})

	target, err := Archive(root, "AI-001-x", ArchiveOptions{PRURL: "https://example/pr/9", Date: "2026-06-13"})
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	got := mustRead(t, filepath.Join(target, "proposal.md"))
	if !strings.Contains(got, "<!-- archived 2026-06-13 — PR: https://example/pr/9 -->") {
		t.Errorf("PR provenance comment missing or malformed:\n%s", got)
	}
}
