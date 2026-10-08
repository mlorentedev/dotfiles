package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedProvenanceSpec writes a spec dir holding review.md with the given frontmatter body.
func seedProvenanceSpec(t *testing.T, reviewBody string) string {
	t.Helper()
	dir := t.TempDir()
	if reviewBody != "" {
		if err := os.WriteFile(filepath.Join(dir, ReviewFile), []byte(reviewBody), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const provenanceReviewDoc = `---
spec: "X"
verdict: "PASS"
reviewed_sha: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-22"
---

Body.
`

// TestProvenanceCatchesAReviewThatWroteNothing is #1157, and both halves of it
// were measured on 2026-08-22 within an hour of each other:
//
//   - the pi primary ran 51 tool executions, settled, and wrote no verdict;
//   - the agy fallback returned `{"status":"ERROR","error":"Individual quota
//     reached"}` and wrote no verdict.
//
// In both cases `dotf spec review` exited 0 — correctly, since a detached launch
// can only report that the runner STARTED — and the PREVIOUS round's review.md
// was still on disk, carrying a PASS and a sha from a tree that no longer
// existed. A stale verdict left in place is indistinguishable from a fresh one
// that reached the same conclusion, and `spec archive` would have accepted it.
//
// The digest is what makes this runner-agnostic. Parsing the transcript would
// need a schema per runner — pi emits `{"type":"agent_settled"}`, agy emits
// `{"event":"result",...}` — and lesson 215 records exactly that parser reading
// the other runner's output as empty. "Did the file change?" needs no schema.
func TestProvenanceCatchesAReviewThatWroteNothing(t *testing.T) {
	dir := seedProvenanceSpec(t, provenanceReviewDoc)
	if err := WriteReviewRequest(dir, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "nan/deepseek-v4-flash", "basebasebase"); err != nil {
		t.Fatal(err)
	}
	// The reviewer writes nothing: review.md is byte-identical to launch time.
	err := checkReviewProvenance(dir, Review{
		Verdict:     "PASS",
		ReviewedSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Reviewer:    "nan/deepseek-v4-flash",
	})
	if err == nil {
		t.Fatal("a review.md unchanged since launch is the PREVIOUS round's verdict; archiving on it is the #1157 failure")
	}
	if !strings.Contains(err.Error(), "wrote no verdict") {
		t.Errorf("the error must name the cause, not a symptom: %v", err)
	}
}

// TestProvenanceAcceptsAFreshVerdict pins the other direction, so the guard
// cannot become one that refuses everything.
func TestProvenanceAcceptsAFreshVerdict(t *testing.T) {
	dir := seedProvenanceSpec(t, "old content")
	if err := WriteReviewRequest(dir, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "nan/deepseek-v4-flash", "basebasebase"); err != nil {
		t.Fatal(err)
	}
	// The reviewer overwrites review.md with a verdict on the launched sha.
	if err := os.WriteFile(filepath.Join(dir, ReviewFile), []byte(provenanceReviewDoc), 0o600); err != nil {
		t.Fatal(err)
	}
	err := checkReviewProvenance(dir, Review{
		Verdict:     "PASS",
		ReviewedSHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Reviewer:    "nan/deepseek-v4-flash",
	})
	if err != nil {
		t.Fatalf("a verdict written after launch, on the launched sha, by the launched reviewer must pass: %v", err)
	}
}

// TestProvenanceCatchesAVerdictOnAnotherSha covers #1153's half: the reviewer
// authors its own `reviewed_sha`, so that field is a CLAIM. The sidecar records
// what the launcher actually pointed it at, which is a measurement.
func TestProvenanceCatchesAVerdictOnAnotherSha(t *testing.T) {
	dir := seedProvenanceSpec(t, "old content")
	if err := WriteReviewRequest(dir, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "nan/deepseek-v4-flash", "basebasebase"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ReviewFile), []byte(provenanceReviewDoc), 0o600); err != nil {
		t.Fatal(err)
	}
	err := checkReviewProvenance(dir, Review{
		Verdict:     "PASS",
		ReviewedSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Reviewer:    "nan/deepseek-v4-flash",
	})
	if err == nil {
		t.Fatal("a verdict claiming a sha the launcher never pointed at must not archive")
	}
	if !strings.Contains(err.Error(), "launched against") {
		t.Errorf("error should contrast the claim with the measurement: %v", err)
	}
}

// TestProvenanceCatchesADifferentPoolMember is the gap checkReviewerPool cannot
// see: both ids are admitted, so only the sidecar knows which one was asked.
func TestProvenanceCatchesADifferentPoolMember(t *testing.T) {
	dir := seedProvenanceSpec(t, "old content")
	if err := WriteReviewRequest(dir, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "agy/gemini-3.1-pro-high", "basebasebase"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ReviewFile), []byte(provenanceReviewDoc), 0o600); err != nil {
		t.Fatal(err)
	}
	err := checkReviewProvenance(dir, Review{
		Verdict:     "PASS",
		ReviewedSHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Reviewer:    "nan/deepseek-v4-flash",
	})
	if err == nil {
		t.Fatal("a verdict signed by a pool member other than the one launched must not archive")
	}
}

// TestProvenanceRefusesAMissingSidecar is #1908: a repository that ignores the
// sidecar loses it on every fresh clone, and a missing sidecar used to skip the
// reviewer and SHA cross-checks silently. It must refuse, and name the remedy.
func TestProvenanceRefusesAMissingSidecar(t *testing.T) {
	dir := seedProvenanceSpec(t, provenanceReviewDoc)
	err := checkReviewProvenance(dir, Review{
		Verdict:     "PASS",
		ReviewedSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Reviewer:    "nan/deepseek-v4-flash",
	})
	if err == nil {
		t.Fatal("a review with no launcher sidecar must not archive: nothing can vouch for its reviewer or SHA")
	}
	for _, want := range []string{ReviewRequestFile, "git check-ignore", "dotf spec review " + filepath.Base(dir)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal does not name %q: %v", want, err)
		}
	}
}

// An empty sidecar is the same skip as a missing one: every cross-check is
// conditional on the field it compares, so `{}` would pass them all (#1908).
func TestProvenanceRefusesAnIncompleteSidecar(t *testing.T) {
	review := Review{
		Verdict:     "PASS",
		ReviewedSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Reviewer:    "nan/deepseek-v4-flash",
	}
	for name, body := range map[string]string{
		"empty":       "{}\n",
		"no reviewer": `{"reviewed_sha": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
		"no sha":      `{"reviewer": "nan/deepseek-v4-flash"}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := seedProvenanceSpec(t, provenanceReviewDoc)
			if err := os.WriteFile(filepath.Join(dir, ReviewRequestFile), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			err := checkReviewProvenance(dir, review)
			if err == nil {
				t.Fatal("a sidecar that names no reviewer or no commit must not vouch for the review")
			}
			if !strings.Contains(err.Error(), "dotf spec review "+filepath.Base(dir)) {
				t.Errorf("refusal does not name the remedy: %v", err)
			}
		})
	}
}

// TestRequestIgnored runs real git: the launcher refuses an ignored request,
// because it would be lost on every other checkout (#1908).
func TestRequestIgnored(t *testing.T) {
	root := t.TempDir()
	gitRun(t, root, "init", "-q", "-b", "main")
	specDir := filepath.Join(root, "specs", "AI-001-x")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if RequestIgnored(root, specDir) {
		t.Fatal("no ignore rule, yet the request reads as ignored")
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("specs/**/"+ReviewRequestFile+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !RequestIgnored(root, specDir) {
		t.Error("the ignore rule matches the request, yet it reads as not ignored")
	}
	if RequestIgnored(t.TempDir(), specDir) {
		t.Error("outside a repository the answer must be false: the archive gate refuses later")
	}
}

// TestUnparseableSidecarIsLoud is C15 for this file: a damaged sidecar must not
// read as "no sidecar", because that drops the guard exactly when the file
// carrying it is broken.
func TestUnparseableSidecarIsLoud(t *testing.T) {
	dir := seedProvenanceSpec(t, provenanceReviewDoc)
	if err := os.WriteFile(filepath.Join(dir, ReviewRequestFile), []byte("{ not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkReviewProvenance(dir, Review{Verdict: "PASS"}); err == nil {
		t.Fatal("a damaged sidecar must fail loudly, not silently disable the guard it carries")
	}
}

// TestWriteReviewRequestRecordsTheDigestOfWhatWasThere pins the field the
// no-verdict check depends on, including the no-previous-review case.
func TestWriteReviewRequestRecordsTheDigestOfWhatWasThere(t *testing.T) {
	t.Run("with a previous review", func(t *testing.T) {
		dir := seedProvenanceSpec(t, provenanceReviewDoc)
		if err := WriteReviewRequest(dir, "sha", "r", "basebasebase"); err != nil {
			t.Fatal(err)
		}
		req, found, err := ReadReviewRequest(dir)
		if err != nil || !found {
			t.Fatalf("read back: %v found=%v", err, found)
		}
		if req.ReviewDigestBefore == "" {
			t.Error("a review.md existed at launch, so its digest must be recorded")
		}
	})

	t.Run("with no previous review", func(t *testing.T) {
		dir := seedProvenanceSpec(t, "")
		if err := WriteReviewRequest(dir, "sha", "r", "basebasebase"); err != nil {
			t.Fatal(err)
		}
		req, _, err := ReadReviewRequest(dir)
		if err != nil {
			t.Fatal(err)
		}
		if req.ReviewDigestBefore != "" {
			t.Error("no review.md at launch must record an empty digest, so the first review is never mistaken for an unchanged one")
		}
	})
}

// VerifyReviewProduced is the launcher's own half of the no-verdict guard, and
// it exists because the archive gate fires too late to help: days on, in another
// session, with the transcript no longer in hand. Every Windows run is a
// foreground run (tmux is Linux-only), so this is the common path there.
func TestVerifyReviewProducedCatchesARunThatWroteNoFile(t *testing.T) {
	dir := seedProvenanceSpec(t, "")
	if err := WriteReviewRequest(dir, "aaaa", "nan/deepseek-v4-flash", "basebasebase"); err != nil {
		t.Fatal(err)
	}

	err := VerifyReviewProduced(dir, "/transcripts/AI-042.jsonl")
	if err == nil {
		t.Fatal("a run that wrote no review.md must fail")
	}
	if !strings.Contains(err.Error(), "/transcripts/AI-042.jsonl") {
		t.Errorf("the error must name the transcript, got %q", err)
	}
}

// The AI-042 round 4 shape exactly: a review.md IS on disk, but it is the
// previous round's, and a reader cannot tell it from a fresh verdict that
// reached the same conclusion.
func TestVerifyReviewProducedCatchesAnUnchangedVerdict(t *testing.T) {
	dir := seedProvenanceSpec(t, provenanceReviewDoc)
	if err := WriteReviewRequest(dir, "aaaa", "nan/deepseek-v4-flash", "basebasebase"); err != nil {
		t.Fatal(err)
	}

	err := VerifyReviewProduced(dir, "/transcripts/AI-042.jsonl")
	if err == nil {
		t.Fatal("an unchanged review.md must fail: the reviewer wrote nothing")
	}
	if !strings.Contains(err.Error(), "byte-identical") {
		t.Errorf("the error must say the file did not move, got %q", err)
	}
}

// WIN-014 produced this shape: the reviewer changed review.md, but wrote no
// machine-readable verdict. A changed digest proves activity, not a valid
// review, so the foreground launcher must fail before reporting success.
func TestVerifyReviewProducedCatchesAMalformedVerdict(t *testing.T) {
	dir := seedProvenanceSpec(t, "old content")
	if err := WriteReviewRequest(dir, "aaaa", "agy/gemini-3.1-pro-high", "basebasebase"); err != nil {
		t.Fatal(err)
	}
	malformed := `---
reviewer: agy/gemini-3.1-pro-high
reviewed_sha: aaaa
---

FAIL
`
	if err := os.WriteFile(filepath.Join(dir, ReviewFile), []byte(malformed), 0o600); err != nil {
		t.Fatal(err)
	}

	err := VerifyReviewProduced(dir, "/transcripts/WIN-014.jsonl")
	if err == nil {
		t.Fatal("a changed review.md without a verdict must fail")
	}
	for _, want := range []string{"review.md", "verdict", "/transcripts/WIN-014.jsonl"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got %q", want, err)
		}
	}
}

func TestVerifyReviewProducedAcceptsAFreshVerdict(t *testing.T) {
	dir := seedProvenanceSpec(t, provenanceReviewDoc)
	if err := WriteReviewRequest(dir, "aaaa", "nan/deepseek-v4-flash", "basebasebase"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ReviewFile), []byte(provenanceReviewDoc+"\nA new round.\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := VerifyReviewProduced(dir, "/transcripts/AI-042.jsonl"); err != nil {
		t.Errorf("a verdict written during the run must pass, got %v", err)
	}
}

// Without a sidecar there is no digest to compare against, but the artifact's
// own schema is still observable. A valid hand-written review therefore passes
// even though launch provenance cannot be asserted.
func TestVerifyReviewProducedAcceptsAValidVerdictWithoutASidecar(t *testing.T) {
	dir := seedProvenanceSpec(t, provenanceReviewDoc)
	if err := VerifyReviewProduced(dir, "/transcripts/AI-042.jsonl"); err != nil {
		t.Errorf("no sidecar must not be an error, got %v", err)
	}
}

func TestVerifyReviewProducedCatchesMalformedVerdictWithoutASidecar(t *testing.T) {
	dir := seedProvenanceSpec(t, "---\nreviewer: nan/x\nreviewed_sha: aaaa\n---\nFAIL\n")

	err := VerifyReviewProduced(dir, "/transcripts/no-sidecar.jsonl")
	if err == nil {
		t.Fatal("a malformed review.md must fail even when provenance is unavailable")
	}
	for _, want := range []string{"review.md", "verdict", "/transcripts/no-sidecar.jsonl"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got %q", want, err)
		}
	}
}

// TestWriteReviewRequestReportsAFailedWrite is GUARD-005 AC8's first half: a
// sidecar that cannot be written is an error the launcher sees, never a silent
// success. The launcher turns it into a warning and still launches, because
// refusing to review over a sidecar would make the guard a liability. A path
// that is a file, not a directory, fails on every OS and under root, where a
// read-only directory does not.
func TestWriteReviewRequestReportsAFailedWrite(t *testing.T) {
	notADir := filepath.Join(t.TempDir(), "spec")
	if err := os.WriteFile(notADir, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := WriteReviewRequest(notADir, "abc", "nan/x", "")
	if err == nil || !strings.Contains(err.Error(), ReviewRequestFile) {
		t.Fatalf("want an error naming %s, got %v", ReviewRequestFile, err)
	}
}
