package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BUG-168 (#1511): a run that ended without a verdict leaves a review file
// saying so, and the gate refuses it by name rather than as unrecognized.
func TestParseReviewRefusesAnErrorVerdictByName(t *testing.T) {
	content := "---\nspec: \"BUG-168-x\"\nverdict: ERROR\nreviewer: \"agy/gemini\"\nreviewed_sha: \"abc\"\n---\n\nquota reached\n"
	r, err := ParseReview(content)
	if err == nil {
		t.Fatal("an ERROR review must be refused")
	}
	if r.Verdict != VerdictError || !r.Verdict.Blocks() {
		t.Errorf("verdict %q must parse as ERROR and block", r.Verdict)
	}
	for _, want := range []string{"ended without a verdict", "ERROR", "run the review again"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must name %q, got: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "unrecognized") {
		t.Errorf("ERROR is a known verdict, not an unrecognized one: %v", err)
	}
}

// stubSpec is a spec folder holding a launch request for slot, and the
// transcript path the run wrote to.
func stubSpec(t *testing.T, slot ReviewSlot) (review, transcript string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "specs", "BUG-168-x")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	req := `{"reviewed_sha":"0123456789abcdef","reviewer":"agy/gemini-3.1-pro-high"}`
	if err := os.WriteFile(filepath.Join(dir, slot.Request), []byte(req), 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, slot.Review), filepath.Join(dir, slot.Transcript)
}

// AC1: the stub carries what the request recorded, how the run ended and what
// the reviewer itself said, and it reads back as an ERROR review of that commit.
func TestWriteErrorStubRecordsTheRunThatSignedNothing(t *testing.T) {
	for _, slot := range []ReviewSlot{FirstSigner, SecondSigner} {
		t.Run(slot.Name, func(t *testing.T) {
			review, transcript := stubSpec(t, slot)
			quota := `{"event":"result","result":{"status":"ERROR","error":"Individual quota reached."}}`
			if err := os.WriteFile(transcript, []byte(quota+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			wrote, err := WriteErrorStub(review, transcript, "the reviewer exited 0 without writing "+slot.Review)
			if err != nil || !wrote {
				t.Fatalf("wrote=%v err=%v, want a stub", wrote, err)
			}
			raw, err := os.ReadFile(review)
			if err != nil {
				t.Fatal(err)
			}
			got := string(raw)
			r, perr := ParseReview(got)
			if r.Verdict != VerdictError || perr == nil {
				t.Fatalf("verdict %q (err %v), want a refused ERROR:\n%s", r.Verdict, perr, got)
			}
			if r.Reviewer != "agy/gemini-3.1-pro-high" || r.ReviewedSHA != "0123456789abcdef" || r.Spec != "BUG-168-x" {
				t.Errorf("frontmatter %+v does not carry the request's reviewer, commit and spec", r)
			}
			for _, want := range []string{"exited 0 without writing", "Individual quota reached.", "dotf spec review BUG-168-x" + slot.Flag()} {
				if !strings.Contains(got, want) {
					t.Errorf("the stub must contain %q:\n%s", want, got)
				}
			}
		})
	}
}

// AC2: a file the run left, signed or provisional, is the reviewer's own and is
// never replaced.
func TestWriteErrorStubLeavesAnExistingReviewFile(t *testing.T) {
	for _, body := range []string{"---\nverdict: PENDING\n---\nfindings so far\n", "---\nverdict: PASS\nreviewed_sha: abc\n---\n"} {
		review, transcript := stubSpec(t, FirstSigner)
		if err := os.WriteFile(review, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		wrote, err := WriteErrorStub(review, transcript, "exited 0")
		if err != nil || wrote {
			t.Errorf("wrote=%v err=%v over an existing file", wrote, err)
		}
		if raw, _ := os.ReadFile(review); string(raw) != body {
			t.Errorf("the reviewer's file was changed:\n%s", raw)
		}
	}
}

// A run that wrote nothing anywhere still gets a stub, which says so, and one
// without a launch request still names the spec.
func TestWriteErrorStubWithoutOutputOrRequest(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "specs", "BUG-168-x")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	review := filepath.Join(dir, ReviewFile)
	if _, err := WriteErrorStub(review, filepath.Join(dir, TranscriptFile), "the reviewer exited with status 3"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(review)
	for _, want := range []string{"verdict: ERROR", "status 3", "wrote nothing"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("the stub must contain %q:\n%s", want, raw)
		}
	}
}

// AC4: a stopped run whose file is the stub says so instead of "not rewritten".
func TestStoppedRunRecordNamesAnErrorStub(t *testing.T) {
	review, transcript := stubSpec(t, FirstSigner)
	if _, err := WriteErrorStub(review, transcript, "stopped at the 45m deadline"); err != nil {
		t.Fatal(err)
	}
	got := StoppedRunRecord(review)
	if !strings.Contains(got, "verdict ERROR") || strings.Contains(got, "cannot be read") {
		t.Errorf("StoppedRunRecord = %q, want it to name the ERROR record", got)
	}
}

// AC3: a foreground run that ends in the stub reports a failed run, never a
// malformed verdict the reviewer wrote, and points at the stub's body.
func TestVerifyReviewProducedReportsAnErrorStub(t *testing.T) {
	review, transcript := stubSpec(t, FirstSigner)
	if _, err := WriteErrorStub(review, transcript, "the reviewer exited with status 3"); err != nil {
		t.Fatal(err)
	}
	err := VerifyReviewProducedIn(filepath.Dir(review), FirstSigner, transcript)
	if err == nil {
		t.Fatal("an ERROR stub must not verify as a produced review")
	}
	if strings.Contains(err.Error(), "malformed") || !strings.Contains(err.Error(), "ended without a verdict") {
		t.Errorf("err = %v, want the ERROR refusal and no 'malformed'", err)
	}
}
