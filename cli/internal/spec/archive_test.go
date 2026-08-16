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

// #998. The gate demanded a review, and the review's own output then failed
// the gate: the adversarial-review skill tells the reviewer to check the spec
// for these markers, so checking writes the literals into review.md and into the
// transcript, both of which sit in the scanned folder. Observed on HARNESS-072:
// 1 hit in review.md (the row certifying the spec was clean) and 3593 in the
// transcript, while every authored artifact was genuinely clean.
func TestFindUnresolvedTagsSkipsReviewMachineryOutput(t *testing.T) {
	root := t.TempDir()
	dir := writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: implementing\n---\nno tags here\n",
		"tasks.md":    "- [x] done\n",
		// The reviewer certifying the spec is clean is what used to make the
		// scan call it dirty.
		ReviewFile: "---\nverdict: \"PASS\"\n---\n| No [AGENT-DRAFT] tags | OK | none found |\n",
		// The reviewer's raw event stream, carrying the literal mid-thought.
		TranscriptFile: `{"type":"message_update","delta":"look for [AGENT-DRAFT] or [AGENT-SUGGESTION] tags"}` + "\n",
	})
	if err := os.WriteFile(filepath.Join(dir, StderrPath(TranscriptFile)),
		[]byte("warn: [AGENT-DRAFT]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tags, err := FindUnresolvedTags(dir)
	if err != nil {
		t.Fatalf("FindUnresolvedTags: %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("review machinery output must not block an archive, got %d hits: %v", len(tags), tags)
	}
}

// The exemption is by exact name, not by shape: a live marker in an artifact the
// author added later must still refuse. Without this, "skip the review's files"
// could quietly widen into "skip files that look machine-generated".
func TestFindUnresolvedTagsStillScansOtherFiles(t *testing.T) {
	root := t.TempDir()
	dir := writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md":   "clean\n",
		ReviewFile:      "| No [AGENT-DRAFT] tags | OK |\n",
		TranscriptFile:  `{"delta":"[AGENT-DRAFT]"}` + "\n",
		"design.md":     "<!-- [AGENT-DRAFT] pick the storage engine -->\n",
		"notes.jsonl":   `{"note":"[AGENT-SUGGESTION] rename this"}` + "\n",
		"review-old.md": "<!-- [AGENT-SUGGESTION] stale copy -->\n",
	})

	tags, err := FindUnresolvedTags(dir)
	if err != nil {
		t.Fatalf("FindUnresolvedTags: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("want 3 hits (design.md, notes.jsonl, review-old.md), got %d: %v", len(tags), tags)
	}
	joined := strings.Join(tags, "\n")
	for _, want := range []string{"design.md", "notes.jsonl", "review-old.md"} {
		if !strings.Contains(joined, want) {
			t.Errorf("tags output missing %q:\n%s", want, joined)
		}
	}
	for _, unwanted := range []string{ReviewFile, TranscriptFile} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("%s must be skipped, but appears in:\n%s", unwanted, joined)
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
		"review.md":   passingReview("AI-001-x"),
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
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: draft\n---\n",
		"review.md":   passingReview("AI-001-x"),
	})

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
		"review.md":   passingReview("AI-001-x"),
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
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md": "---\nstatus: draft\n---\nbody\n",
		"review.md":   passingReview("AI-001-x"),
	})

	target, err := Archive(root, "AI-001-x", ArchiveOptions{PRURL: "https://example/pr/9", Date: "2026-06-13"})
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	got := mustRead(t, filepath.Join(target, "proposal.md"))
	if !strings.Contains(got, "<!-- archived 2026-06-13 — PR: https://example/pr/9 -->") {
		t.Errorf("PR provenance comment missing or malformed:\n%s", got)
	}
}

// BUG-041. The pre-flight was inverted: it matched the bare `[AGENT-DRAFT]`
// shape, which this repo writes when DOCUMENTING the markers, and missed the
// suffixed shape /spec fill actually emits — so it cried wolf on prose and let a
// live tag into the archive (specs/archive/CLI-002-repo-structure/proposal.md).
func TestScanUnresolvedTags(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []int
	}{
		// --- must NOT fire: the tag is quoted, not live ---
		{
			name:    "inside an inline code span",
			content: "- [ ] Add `sb_specs` (active/archived counts + `[AGENT-DRAFT]` flagging)\n",
		},
		{
			name:    "on a completed checklist item",
			content: "- [x] No open questions left — the Linux [AGENT-DRAFT] is resolved\n",
		},
		{
			name:    "inside a fenced code block",
			content: "before\n```console\n$ dotf spec archive X\nError: [AGENT-DRAFT] found\n```\nafter\n",
		},
		{
			name:    "inside a tilde-fenced block",
			content: "~~~\n[AGENT-SUGGESTION — accept or remove]\n~~~\n",
		},
		{
			// CommonMark requires the opening and closing runs to be equal, so
			// this is NOT a code span. Stripping it anyway would hide a live
			// marker -- the false-negative direction a guard must never take.
			name:    "mismatched backtick runs are not a code span",
			content: "a `[AGENT-DRAFT]`` b\n",
			want:    []int{1},
		},
		{
			name:    "an unterminated backtick run leaves the line scannable",
			content: "a ` [AGENT-DRAFT] b\n",
			want:    []int{1},
		},
		{
			name:    "a double-backtick span is still a span when balanced",
			content: "a ``[AGENT-DRAFT]`` b\n",
		},
		// --- must fire: the tag is live ---
		{
			name:    "the canonical emitted form in prose",
			content: "Some rationale. [AGENT-SUGGESTION — accept or remove] More text.\n",
			want:    []int{1},
		},
		{
			name:    "the canonical draft form in prose",
			content: "intro\n<!-- [AGENT-DRAFT — review before archive] -->\n",
			want:    []int{2},
		},
		{
			name:    "the bare form in a plain HTML comment",
			content: "line one\n<!-- [AGENT-DRAFT] write the why -->\nline three\n",
			want:    []int{2},
		},
		{
			name:    "on an UNticked checklist item",
			content: "- [ ] do it [AGENT-SUGGESTION] consider Y\n",
			want:    []int{1},
		},
		{
			name:    "after a fenced block has closed",
			content: "```\n[AGENT-DRAFT]\n```\n[AGENT-DRAFT — review before archive]\n",
			want:    []int{4},
		},
		{
			name:    "a longer closing fence still closes the block",
			content: "```\nquoted [AGENT-DRAFT]\n````\nlive [AGENT-DRAFT]\n",
			want:    []int{4},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ScanUnresolvedTags(tc.content)
			if len(got) != len(tc.want) {
				t.Fatalf("lines %v, want %v\ncontent:\n%s", got, tc.want, tc.content)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("lines %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// The regression this fix exists to prevent, at the level the user experiences
// it: a spec documenting the markers must archive with no --force-with-drafts.
func TestArchiveAcceptsSpecThatOnlyQuotesTags(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-002-y", map[string]string{
		"proposal.md":     "---\nstatus: implementing\n---\nprose\n",
		"tasks.md":        "- [x] Add `sb_specs` (`[AGENT-DRAFT]` flagging — lifted from detect_repo_specs)\n",
		"verification.md": "```\n[AGENT-SUGGESTION]\n```\n",
		"review.md":       passingReview("AI-002-y"),
	})

	if _, err := Archive(root, "AI-002-y", ArchiveOptions{}); err != nil {
		t.Fatalf("archive refused a spec that only quotes the markers: %v", err)
	}
}

// The red direction. Without this, a scanner that matched nothing at all would
// satisfy every case above.
func TestArchiveStillRefusesTheEmittedTagForm(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-003-z", map[string]string{
		"proposal.md":     "---\nstatus: implementing\n---\nWhy: [AGENT-DRAFT — review before archive]\n",
		"tasks.md":        "- [ ] do it\n",
		"verification.md": "clean\n",
	})

	_, err := Archive(root, "AI-003-z", ArchiveOptions{})
	if err == nil {
		t.Fatal("archive accepted a spec carrying the canonical emitted tag form")
	}
	if !strings.Contains(err.Error(), "AGENT-DRAFT") {
		t.Fatalf("refusal does not name the tag: %v", err)
	}
}
