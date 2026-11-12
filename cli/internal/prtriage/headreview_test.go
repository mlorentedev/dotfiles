package prtriage

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

const (
	reviewRan  = `{"check_runs":[{"name":"review","status":"completed","conclusion":"success","started_at":"2026-11-12T08:00:00Z"}]}`
	headSHA    = "0123456789abcdef"
	openPR10   = `{"head":{"sha":"` + headSHA + `"},"number":10,"title":"mine","html_url":"u10","state":"open"}`
	openDraft  = `{"head":{"sha":"` + headSHA + `"},"number":10,"title":"mine","html_url":"u10","state":"open","draft":true}`
	checksPath = "/commits/" + headSHA + "/check-runs"
)

func checkRuns(runs ...string) string {
	return `{"check_runs":[` + strings.Join(runs, ",") + `]}`
}

func checkRun(name, status, conclusion, started string) string {
	return fmt.Sprintf(`{"name":%q,"status":%q,"conclusion":%q,"started_at":%q}`, name, status, conclusion, started)
}

// TestReviewOnHeadHoldsAQueueTheCommentsLeaveClear is #3243 AC1: with nothing
// to triage, a head whose review did not run is not a clear queue. That is the
// ts-bridge#687 shape: the review failed and published nothing, the queue read
// clear, and the head landed unreviewed.
func TestReviewOnHeadHoldsAQueueTheCommentsLeaveClear(t *testing.T) {
	tests := []struct {
		name    string
		runs    string
		reading string // "" means the PR is clear
	}{
		{"failed", checkRuns(checkRun("review", "completed", "failure", "2026-11-12T08:00:00Z")), "failure"},
		{"skipped", checkRuns(checkRun("review", "completed", "skipped", "2026-11-12T08:00:00Z")), "skipped"},
		{"still running", checkRuns(checkRun("review", "in_progress", "", "2026-11-12T08:00:00Z")), "in_progress"},
		{"absent", checkRuns(checkRun("lint", "completed", "success", "2026-11-12T08:00:00Z")), ReviewAbsent},
		{"the reusable workflow's name, failed", checkRuns(checkRun("review / review", "completed", "failure", "2026-11-12T08:00:00Z")), "failure"},
		{"cancelled, then a success on the same head", checkRuns(
			checkRun("review", "completed", "cancelled", "2026-11-12T08:00:00Z"),
			checkRun("review", "completed", "success", "2026-11-12T08:05:00Z")), ""},
		{"a success, then a failure on the same head", checkRuns(
			checkRun("review / review", "completed", "success", "2026-11-12T08:00:00Z"),
			checkRun("review / review", "completed", "failure", "2026-11-12T08:05:00Z")), "failure"},
		{"a check merely named like it", checkRuns(checkRun("preview", "completed", "failure", "2026-11-12T08:00:00Z"),
			checkRun("review", "completed", "success", "2026-11-12T08:00:00Z")), ""},
	}
	for _, tc := range tests {
		for _, scope := range []string{"repo-wide", "--pr"} {
			t.Run(tc.name+", "+scope, func(t *testing.T) {
				fake := &fakeGH{responses: map[string]string{
					"pulls?state": "[" + openPR10 + "]", "/pulls/10": openPR10,
					"/issues/10/comments": `[]`, checksPath: tc.runs,
				}}
				var got []Status
				var err error
				if scope == "--pr" {
					got, err = fetchPRWith(context.Background(), fake.run, "o/r", testRegistry(), 10)
				} else {
					got, err = fetchWith(context.Background(), fake.run, "o/r", testRegistry())
				}
				if err != nil {
					t.Fatalf("fetch: %v", err)
				}
				if tc.reading == "" {
					if len(got) != 0 {
						t.Fatalf("want a clear queue, got %+v", got)
					}
					return
				}
				if len(got) != 1 || !got[0].Pending || !got[0].HeadUnreviewed {
					t.Fatalf("want #10 held, got %+v", got)
				}
				for _, want := range []string{"no review covers the head", "0123456", "reads " + tc.reading} {
					if !strings.Contains(got[0].Reason, want) {
						t.Errorf("reason %q lacks %q", got[0].Reason, want)
					}
				}
			})
		}
	}
}

// AC2: a draft is not asking to land, so its head review holds nothing, and
// its check runs are never read.
func TestReviewOnHeadIgnoresADraft(t *testing.T) {
	fake := &fakeGH{responses: map[string]string{
		"pulls?state": "[" + openDraft + "]", "/issues/10/comments": `[]`,
		checksPath: checkRuns(checkRun("review", "completed", "skipped", "2026-11-12T08:00:00Z")),
	}}
	got, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("a draft was held: %+v", got)
	}
	for _, r := range fake.seen() {
		if strings.Contains(r, "/check-runs") {
			t.Errorf("a draft's check runs were read: %s", r)
		}
	}
}

// AC2: a release PR skips the reviewer by design, and its diff is an exempt
// signature, so it stays quiet. A diff that is not exempt is held, which is
// what keeps the exemption from becoming a way past the rule.
func TestReviewOnHeadLetsAnExemptDiffThrough(t *testing.T) {
	reg := testRegistry()
	reg.Exempt.Signatures = []Signature{{Name: "release-please", Files: []string{"CHANGELOG.md", "versions.conf"}}}
	for _, tc := range []struct {
		files string
		held  bool
	}{
		{`[{"filename":"CHANGELOG.md"},{"filename":"versions.conf"}]`, false},
		{`[{"filename":"CHANGELOG.md"},{"filename":"cli/main.go"}]`, true},
	} {
		fake := &fakeGH{responses: map[string]string{
			"pulls?state": "[" + openPR10 + "]", "/issues/10/comments": `[]`, "/pulls/10/files": tc.files,
			checksPath: checkRuns(checkRun("review", "completed", "skipped", "2026-11-12T08:00:00Z")),
		}}
		got, err := fetchWith(context.Background(), fake.run, "o/r", reg)
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		if held := len(got) == 1; held != tc.held {
			t.Errorf("files %s: held = %v, want %v (%+v)", tc.files, held, tc.held, got)
		}
	}
}

// A head whose check runs cannot be read is an error, never a clear queue.
func TestReviewOnHeadFailsWhenTheChecksCannotBeRead(t *testing.T) {
	for name, fake := range map[string]*fakeGH{
		"the read fails": {responses: map[string]string{"pulls?state": "[" + openPR10 + "]", "/issues/10/comments": `[]`},
			errs: map[string]error{"/check-runs": fmt.Errorf("gh: HTTP 502")}},
		"a full page": {responses: map[string]string{"pulls?state": "[" + openPR10 + "]", "/issues/10/comments": `[]`,
			checksPath: checkRuns(strings.Repeat(checkRun("lint", "completed", "success", "2026-11-12T08:00:00Z")+",", checkRunLimit-1) +
				checkRun("lint", "completed", "success", "2026-11-12T08:00:00Z"))}},
		"no head commit": {responses: map[string]string{"pulls?state": `[{"number":10,"title":"mine","html_url":"u10","state":"open"}]`, "/issues/10/comments": `[]`}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
			if err == nil {
				t.Fatalf("want an error, got the queue %+v", got)
			}
		})
	}
}

func TestIsReviewCheck(t *testing.T) {
	for name, want := range map[string]bool{
		"review": true, "review / review": true, "pr-agent / review": true,
		"preview": false, "review-attestation": false, "review / lint": false, "Review": false, "": false,
	} {
		if got := IsReviewCheck(name); got != want {
			t.Errorf("IsReviewCheck(%q) = %v, want %v", name, got, want)
		}
	}
}
