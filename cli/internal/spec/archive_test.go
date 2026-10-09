package spec

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// answeredPromotions is a verification.md whose promotion candidates are all
// answered, so a fixture about something else passes the promotion pre-flight.
const answeredPromotions = "## Promotion candidates\n\n" +
	"- [x] Lesson for the repo's `docs/lessons/`? no: a fixture\n" +
	"- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a fixture\n" +
	"- [x] New pattern candidate for `00_meta/patterns/`? no: a fixture\n"

// writeSpec materializes specs/<id>/ under root with the given files. A spec
// given no verification.md gets answeredPromotions. A spec given a review.md
// and no review-request.json gets one matching it (fixtureRequest): the archive
// refuses a review without one (#1908), and a fixture about something else
// should not trip on that.
// fixtureRequest is the sidecar the launcher would have written for review: the
// same reviewer and reviewed_sha, no digests. A review that does not parse gets
// placeholders; the gate refuses it before it reads the request.
func fixtureRequest(review string) string {
	sha, reviewer := strings.Repeat("0", 40), "fixture/reviewer"
	if r, err := ParseReview(review); err == nil {
		if r.ReviewedSHA != "" {
			sha = r.ReviewedSHA
		}
		if r.Reviewer != "" {
			reviewer = r.Reviewer
		}
	}
	return fmt.Sprintf("{\"reviewed_sha\": %q, \"reviewer\": %q}\n", sha, reviewer)
}

func writeSpec(t *testing.T, root, id string, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(root, "specs", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := files["verification.md"]; !ok {
		files["verification.md"] = answeredPromotions
	}
	if _, review := files[ReviewFile]; review {
		if _, ok := files[ReviewRequestFile]; !ok {
			files[ReviewRequestFile] = fixtureRequest(files[ReviewFile])
		}
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
			name:      "CRLF fences (CLI-095): value replaced, line endings kept",
			in:        "---\r\nstatus: implementing # x\r\n---\r\nbody\r\n",
			newStatus: "archived",
			want:      "---\r\nstatus: archived # x\r\n---\r\n",
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

// CLI-095 (#1976): a CRLF checkout (Windows, core.autocrlf) carries "---\r"
// fences. The archive reported `status: archived` and left the frontmatter on
// its old token. Every lifecycle token, both line endings, end to end.
func TestArchiveRewritesEveryStatusTokenWithEitherLineEnding(t *testing.T) {
	for _, eol := range []string{"\n", "\r\n"} {
		for _, from := range []string{"draft", "implementing", "verifying", "archived"} {
			root := t.TempDir()
			proposal := strings.Join([]string{"---", "id: AI-001-x",
				"status: " + from + " # draft | implementing | verifying | archived", "---", "# AI-001-x", ""}, eol)
			writeSpec(t, root, "AI-001-x", map[string]string{
				"proposal.md": proposal,
				"review.md":   passingReview("AI-001-x"),
			})
			target, err := Archive(root, "AI-001-x", ArchiveOptions{Abandoned: from == "archived"})
			if err != nil {
				t.Fatalf("eol=%q from=%s: Archive: %v", eol, from, err)
			}
			want := "archived"
			if from == "archived" {
				want = "abandoned"
			}
			got := mustRead(t, filepath.Join(target, "proposal.md"))
			if !strings.Contains(got, eol+"status: "+want+" # draft | implementing | verifying | archived"+eol) {
				t.Errorf("eol=%q from=%s: frontmatter not rewritten to %s:\n%q", eol, from, want, got)
			}
		}
	}
}

// The report must not claim a rewrite that did not happen: a proposal with no
// status: line in its frontmatter, or no frontmatter at all, gets one.
func TestArchiveRecordsAStatusWhereTheProposalHadNone(t *testing.T) {
	for name, proposal := range map[string]string{
		"no frontmatter":              "# AI-001-x\nstatus: draft\n",
		"frontmatter, no status":      "---\nid: AI-001-x\n---\n# AI-001-x\n",
		"CRLF frontmatter, no status": "---\r\nid: AI-001-x\r\n---\r\n# AI-001-x\r\n",
		"CRLF, no frontmatter":        "# AI-001-x\r\nstatus: draft\r\n",
		"thematic break in the body":  "# AI-001-x\n\nintro\n\n---\n\nstatus: draft\n",
	} {
		root := t.TempDir()
		writeSpec(t, root, "AI-001-x", map[string]string{
			"proposal.md": proposal,
			"review.md":   passingReview("AI-001-x"),
		})
		target, err := Archive(root, "AI-001-x", ArchiveOptions{Date: "2026-10-07", PRURL: "https://example.test/pr/1"})
		if err != nil {
			t.Fatalf("%s: Archive: %v", name, err)
		}
		got := mustRead(t, filepath.Join(target, "proposal.md"))
		lines := strings.Split(got, "\n")
		i := frontmatterStatusLine(lines)
		if strings.TrimSpace(lines[0]) != "---" || i < 0 || strings.TrimSpace(lines[i]) != "status: archived" {
			t.Errorf("%s: no frontmatter status: archived:\n%q", name, got)
		}
		if strings.Contains(proposal, "\r\n") && strings.Count(got, "\n") != strings.Count(got, "\r\n") {
			t.Errorf("%s: mixed line endings after the insertion:\n%q", name, got)
		}
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

	if _, err := Archive(root, "AI-001-x", ArchiveOptions{ForceWithDrafts: true, BypassReason: "test"}); err != nil {
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
		"verification.md": "```\n[AGENT-SUGGESTION]\n```\n" + answeredPromotions,
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

// AC-1.3 (#1625 W1.3). Every file the review machinery writes into a spec
// folder is review STATE, not an authored artifact: the draft-tag scan skips
// it, and W3.6's byte-bound review excludes it from the reviewed tree. One
// declared set serves both, so the two cannot drift apart. review-request.json
// is written at launch, after reviewed_sha is fixed; hashing it into the
// reviewed tree would make every review stale on arrival.
func TestDraftScanSkipsEveryReviewStateFile(t *testing.T) {
	want := []string{ReviewFile, TranscriptFile, StderrPath(TranscriptFile), ReviewRequestFile}
	// The second signature (AI-045) writes the same four files under its own names.
	want = append(want, SecondSigner.Review, SecondSigner.Transcript, StderrPath(SecondSigner.Transcript), SecondSigner.Request)
	if len(ReviewStateFiles) != len(want) {
		t.Fatalf("ReviewStateFiles = %v, want exactly %v", ReviewStateFiles, want)
	}
	files := map[string]string{"proposal.md": "clean\n"}
	for _, name := range want {
		if !IsReviewState(name) {
			t.Errorf("%s is written by the review machinery but is not declared review state", name)
		}
		files[name] = "[AGENT-DRAFT] written by the reviewer, not the author\n"
	}
	dir := writeSpec(t, t.TempDir(), "AI-001-x", files)
	tags, err := FindUnresolvedTags(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 0 {
		t.Fatalf("review state must never block an archive, got %v", tags)
	}
}

// The other half of AC-1.3: excluding review state must not blind the scan to
// the author's own contract files.
func TestDraftTagInProposalStillBlocksAfterReview(t *testing.T) {
	dir := writeSpec(t, t.TempDir(), "AI-001-x", map[string]string{
		"proposal.md":     "<!-- [AGENT-DRAFT] decide the retry budget -->\n",
		ReviewFile:        "| No [AGENT-DRAFT] tags | OK |\n",
		ReviewRequestFile: `{"reviewer":"nan/mimo-v2.5"}` + "\n",
	})
	tags, err := FindUnresolvedTags(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || !strings.HasPrefix(tags[0], "proposal.md:") {
		t.Fatalf("want exactly the proposal.md hit, got %v", tags)
	}
	for _, name := range []string{"proposal.md", "tasks.md", "verification.md", "features.json", "design.md", "spec.md"} {
		if IsReviewState(name) {
			t.Errorf("%s is an authored artifact and must not be review state", name)
		}
	}
}

// ReviewStateFiles must be complete, not merely correct (PR-Agent on #1631):
// a new file the review machinery writes into a spec folder, but forgets to
// declare, would be scanned for draft tags and hashed into a reviewed tree,
// failing every archive. So derive completeness from the source rather than
// from convention. Every package constant used as a path component inside the
// spec folder (filepath.Join(specDir, X) or filepath.Join(…, specID, X)) must
// be declared review state, or be one of the files the author writes.
func TestDraftReviewStateListIsCompleteBySource(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	consts := map[string]string{} // package-level string constants
	var joined []string           // identifiers joined under a spec folder
	var literals []string         // string literals joined under a spec folder
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ValueSpec:
				for i, name := range x.Names {
					if i < len(x.Values) {
						if lit, ok := x.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							consts[name.Name] = strings.Trim(lit.Value, "\"`")
						}
					}
				}
			case *ast.CallExpr:
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Join" && len(x.Args) >= 2 {
					underSpec := false
					for _, a := range x.Args[:len(x.Args)-1] {
						if id, ok := a.(*ast.Ident); ok && (id.Name == "specDir" || id.Name == "specID") {
							underSpec = true
						}
					}
					switch last := x.Args[len(x.Args)-1].(type) {
					case *ast.Ident:
						if underSpec {
							joined = append(joined, last.Name)
						}
					case *ast.BasicLit:
						// A path spelled as a literal must not slip past the
						// check that a named constant would face (PR-Agent on #1631).
						if underSpec && last.Kind == token.STRING {
							literals = append(literals, strings.Trim(last.Value, "\"`"))
						}
					}
				}
			}
			return true
		})
	}
	authored := map[string]bool{"proposal.md": true, "tasks.md": true, "verification.md": true, "features.json": true, "spec.md": true}
	checked := 0
	names := literals
	for _, id := range joined {
		if value, isConst := consts[id]; isConst {
			names = append(names, value) // a loop variable or computed name is skipped
		}
	}
	for _, value := range names {
		checked++
		if !IsReviewState(value) && !authored[value] {
			t.Errorf("%q is joined under a spec folder but is neither review state nor an authored artifact — add it to ReviewStateFiles", value)
		}
	}
	if checked == 0 {
		t.Fatal("found no spec-folder path constants — the source walk is broken, so this test proves nothing")
	}
}

// templateArchiveChecklist is the `## Archive checklist` section of the
// verification.md template `dotf spec init` writes, so a change to its wording
// that the ticker no longer matches fails here rather than in the archive.
func templateArchiveChecklist(t *testing.T) string {
	t.Helper()
	b, err := templatesFS.ReadFile("templates/verification.md")
	if err != nil {
		t.Fatal(err)
	}
	_, section, ok := strings.Cut(string(b), "## Archive checklist")
	if !ok {
		t.Fatal("the verification.md template has no archive checklist")
	}
	return "## Archive checklist" + section
}

// #1990: the archive ticks the checklist items it performed or verified, and
// leaves the one it cannot see (the board ticket) for the human.
func TestArchiveTicksTheChecklistItemsItPerformed(t *testing.T) {
	for _, tc := range []struct {
		name      string
		abandoned bool
		ticked    []string
		unticked  []string
	}{
		{"archived", false,
			[]string{"status: archived", "Folder moved", "Promotions above executed"},
			[]string{"Bitácora board ticket"}},
		// The status and folder items describe the archive route, which an
		// abandoned spec did not take.
		{"abandoned", true,
			[]string{"Promotions above executed"},
			[]string{"status: archived", "Folder moved", "Bitácora board ticket"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeSpec(t, root, "AI-001-x", map[string]string{
				"proposal.md":     "---\nstatus: verifying\n---\n",
				"review.md":       passingReview("AI-001-x"),
				"verification.md": answeredPromotions + "\n" + templateArchiveChecklist(t),
			})
			target, err := Archive(root, "AI-001-x", ArchiveOptions{Abandoned: tc.abandoned})
			if err != nil {
				t.Fatalf("Archive: %v", err)
			}
			got := mustRead(t, filepath.Join(target, "verification.md"))
			for _, item := range tc.ticked {
				if !hasItem(got, "[x]", item) {
					t.Errorf("%q not ticked:\n%s", item, got)
				}
			}
			for _, item := range tc.unticked {
				if !hasItem(got, "[ ]", item) {
					t.Errorf("%q ticked, but the archive did not do it:\n%s", item, got)
				}
			}
		})
	}
}

// A verification.md that exists and cannot be read refuses before anything
// moves. On the full route the promotion preflight already reads it; a
// fast-track spec carries its promotions in spec.md, so the archive's own read
// is the only one. Silently skipping it would archive with the checklist
// contradicting the tree.
func TestArchiveRefusesAnUnreadableVerificationBeforeMoving(t *testing.T) {
	root := t.TempDir()
	dir := writeSpec(t, root, "AI-001-x", map[string]string{
		"spec.md":         "---\nstatus: verifying\n---\n" + answeredPromotions,
		"verification.md": "",
	})
	// A directory in its place: a read error that is not "does not exist", on
	// every OS and as any user.
	v := filepath.Join(dir, "verification.md")
	if err := os.Remove(v); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(v, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Archive(root, "AI-001-x", ArchiveOptions{}); err == nil || !strings.Contains(err.Error(), "verification.md") {
		t.Fatalf("archived past an unreadable verification.md: err=%v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("the spec moved although the archive refused: %v", err)
	}
}

func TestUntickedArchiveChecklistNamesWhatIsLeftForAHuman(t *testing.T) {
	ticked := tickArchiveChecklist(templateArchiveChecklist(t), false)
	left := UntickedArchiveChecklist(ticked)
	if len(left) != 1 || !strings.Contains(left[0], "Bitácora board ticket") {
		t.Errorf("want only the board item left, got %q", left)
	}
	if got := UntickedArchiveChecklist("# x\n- [ ] not in the section\n"); got != nil {
		t.Errorf("an item outside the section was reported: %q", got)
	}
}

func hasItem(content, box, fragment string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "- "+box) && strings.Contains(line, fragment) {
			return true
		}
	}
	return false
}

func TestTickArchiveChecklist(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		// Wordings measured across the archived corpus, not only the template's.
		"corpus wordings": {
			"## Archive checklist\n\n" +
				"- [ ] `proposal.md` frontmatter `status: archived`.\n" +
				"- [ ] Folder moved to `specs/archive/X/`\n" +
				"- [ ] Promotions above executed (none)\n" +
				"- [ ] Vault `11-tasks.md` ticked with PR link.\n",
			"## Archive checklist\n\n" +
				"- [x] `proposal.md` frontmatter `status: archived`.\n" +
				"- [x] Folder moved to `specs/archive/X/`\n" +
				"- [x] Promotions above executed (none)\n" +
				"- [ ] Vault `11-tasks.md` ticked with PR link.\n",
		},
		"only inside the section": {
			"## Tasks\n\n- [ ] Folder moved by hand\n\n## Archive checklist\n\n- [ ] Folder moved\n\n## After\n\n- [ ] Folder moved again\n",
			"## Tasks\n\n- [ ] Folder moved by hand\n\n## Archive checklist\n\n- [x] Folder moved\n\n## After\n\n- [ ] Folder moved again\n",
		},
		"CRLF kept": {
			"## Archive checklist\r\n\r\n- [ ] Folder moved\r\n- [ ] Board ticket\r\n",
			"## Archive checklist\r\n\r\n- [x] Folder moved\r\n- [ ] Board ticket\r\n",
		},
		"already ticked is left alone": {
			"## Archive checklist\n- [x] Folder moved\n",
			"## Archive checklist\n- [x] Folder moved\n",
		},
		"no section": {"- [ ] Folder moved\n", "- [ ] Folder moved\n"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := tickArchiveChecklist(tc.in, false); got != tc.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}
