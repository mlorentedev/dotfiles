package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #2264: a second review keeps the first round's verdict under a numbered name
// instead of overwriting it, and the request it answered goes with it.
func TestRotateReviewRoundKeepsThePreviousVerdict(t *testing.T) {
	dir := writeSpec(t, t.TempDir(), "AI-001-x", map[string]string{
		"proposal.md":     "clean\n",
		ReviewFile:        "verdict: FAIL\n",
		ReviewRequestFile: `{"reviewer":"a"}` + "\n",
	})
	kept, err := RotateReviewRound(dir, FirstSigner)
	if err != nil {
		t.Fatal(err)
	}
	if kept != "review-round1.md" {
		t.Fatalf("kept = %q, want review-round1.md", kept)
	}
	assertFile(t, dir, "review-round1.md", "verdict: FAIL\n")
	assertFile(t, dir, "review-request-round1.json", `{"reviewer":"a"}`+"\n")
	for _, gone := range []string{ReviewFile, ReviewRequestFile} {
		if _, err := os.Stat(filepath.Join(dir, gone)); !os.IsNotExist(err) {
			t.Errorf("%s still holds the previous round after rotation (err=%v)", gone, err)
		}
	}

	// The next round goes one past the highest kept, so a round made by hand
	// before rotation existed (kubelab's review-round1.md) is never overwritten.
	if err := os.WriteFile(filepath.Join(dir, ReviewFile), []byte("verdict: PASS\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if kept, err = RotateReviewRound(dir, FirstSigner); err != nil || kept != "review-round2.md" {
		t.Fatalf("second rotation = %q, %v; want review-round2.md", kept, err)
	}
	assertFile(t, dir, "review-round1.md", "verdict: FAIL\n")
	assertFile(t, dir, "review-round2.md", "verdict: PASS\n")
}

func TestRotateReviewRoundIsANoOpWithoutAVerdict(t *testing.T) {
	dir := writeSpec(t, t.TempDir(), "AI-001-x", map[string]string{
		"proposal.md":     "clean\n",
		ReviewRequestFile: `{"reviewer":"a"}` + "\n",
	})
	kept, err := RotateReviewRound(dir, FirstSigner)
	if err != nil || kept != "" {
		t.Fatalf("RotateReviewRound = %q, %v; want no rotation", kept, err)
	}
	assertFile(t, dir, ReviewRequestFile, `{"reviewer":"a"}`+"\n")
}

// The second signer keeps its own rounds, and never takes the first's number.
func TestRotateReviewRoundKeepsEachSlotApart(t *testing.T) {
	dir := writeSpec(t, t.TempDir(), "AI-001-x", map[string]string{
		"review-round1.md":  "first signer's round\n",
		SecondSigner.Review: "second signer's verdict\n",
	})
	kept, err := RotateReviewRound(dir, SecondSigner)
	if err != nil || kept != "review-second-round1.md" {
		t.Fatalf("RotateReviewRound = %q, %v; want review-second-round1.md", kept, err)
	}
	assertFile(t, dir, "review-round1.md", "first signer's round\n")
}

// A kept round quotes the tag literal like the verdict it was, so the archive
// scan skips it; an author's file that merely looks numbered is still scanned.
func TestDraftScanSkipsKeptRoundsButNotLookalikes(t *testing.T) {
	dir := writeSpec(t, t.TempDir(), "AI-001-x", map[string]string{
		"proposal.md":                       "clean\n",
		"review-round1.md":                  "| No [AGENT-DRAFT] tags | OK |\n",
		"review-request-round3.json":        `{"note":"[AGENT-DRAFT]"}` + "\n",
		"review-second-round2.md":           "| No [AGENT-SUGGESTION] tags | OK |\n",
		"notes-round1.md":                   "[AGENT-DRAFT] the author's own\n",
		"review-roundx.md":                  "[AGENT-DRAFT] not a round number\n",
		"review-round0.md":                  "[AGENT-DRAFT] rounds start at 1\n",
		"review-transcript-round1.jsonl.md": "[AGENT-DRAFT] no such base file\n",
	})
	tags, err := FindUnresolvedTags(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(tags, "\n")
	for _, want := range []string{"notes-round1.md:", "review-roundx.md:", "review-round0.md:", "review-transcript-round1.jsonl.md:"} {
		if !strings.Contains(got, want) {
			t.Errorf("%s is not a kept round and must still be scanned; got %v", want, tags)
		}
	}
	for _, kept := range []string{"review-round1.md", "review-request-round3.json", "review-second-round2.md"} {
		if strings.Contains(got, kept+":") {
			t.Errorf("%s is a kept review round and must not block an archive; got %v", kept, tags)
		}
	}
}

func assertFile(t *testing.T, dir, name, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

// #1722: the archive carries every kept round with the spec, and a round that
// quotes the tag literal does not block it.
func TestArchiveCarriesTheKeptRounds(t *testing.T) {
	root := t.TempDir()
	writeSpec(t, root, "AI-001-x", map[string]string{
		"proposal.md":      "---\nstatus: implementing\n---\n# AI-001-x\n",
		"review.md":        passingReview("AI-001-x"),
		"review-round1.md": "verdict: FAIL\n| No [AGENT-DRAFT] tags | OK |\n",
	})
	target, err := Archive(root, "AI-001-x", ArchiveOptions{})
	if err != nil {
		t.Fatalf("a kept round must not block the archive: %v", err)
	}
	assertFile(t, target, "review-round1.md", "verdict: FAIL\n| No [AGENT-DRAFT] tags | OK |\n")
}
