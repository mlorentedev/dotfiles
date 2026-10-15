package prtriage

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// TestFetchPRScopesTheQueueToOnePullRequest is #2549 AC1 and AC2: with a PR
// number, the answer is that PR's alone. Another PR's untriaged output cannot
// make it pending, the list endpoint is never asked, and a PR that cannot be
// read, or is not open, is an error rather than an empty queue.
func TestFetchPRScopesTheQueueToOnePullRequest(t *testing.T) {
	const (
		open10     = `{"number":10,"title":"mine","html_url":"https://github.com/o/r/pull/10","state":"open"}`
		reviewed   = `[{"user":{"login":"github-actions[bot]"},"body":"## PR Reviewer Guide\nfindings","created_at":"2026-10-15T08:00:00Z"}]`
		triaged    = `[{"user":{"login":"github-actions[bot]"},"body":"## PR Reviewer Guide\nfindings","created_at":"2026-10-15T08:00:00Z"},{"user":{"login":"manu"},"body":"## Review triage\nall disposed","created_at":"2026-10-15T08:10:00Z"}]`
		otherPRs   = `[{"number":10,"title":"mine","html_url":"u10"},{"number":11,"title":"another session's","html_url":"u11"}]`
		pendingOn  = `[{"user":{"login":"coderabbitai[bot]"},"body":"No actionable comments were generated","created_at":"2026-10-15T07:00:00Z"}]`
		wantNoList = "pulls?state"
	)
	tests := []struct {
		name        string
		responses   map[string]string
		errs        map[string]error
		wantPending int
		wantErr     string
	}{
		{
			name:      "pending output on another PR does not make this one pending",
			responses: map[string]string{"/pulls/10": open10, "/issues/10/comments": `[]`, "pulls?state": otherPRs, "/issues/11/comments": pendingOn},
		},
		{
			name:        "untriaged output on this PR is pending",
			responses:   map[string]string{"/pulls/10": open10, "/issues/10/comments": reviewed},
			wantPending: 1,
		},
		{
			name:      "a triage after the review clears it",
			responses: map[string]string{"/pulls/10": open10, "/issues/10/comments": triaged},
		},
		{
			name:    "a PR that cannot be read is an error, not an empty queue",
			errs:    map[string]error{"/pulls/10": fmt.Errorf("gh: HTTP 404: Not Found")},
			wantErr: "#10",
		},
		{
			name:      "its comments failing is an error too",
			responses: map[string]string{"/pulls/10": open10},
			errs:      map[string]error{"/issues/10/comments": fmt.Errorf("gh: exit status 1")},
			wantErr:   "comments for #10",
		},
		{
			name:      "an answer for another number is refused",
			responses: map[string]string{"/pulls/10": `{"number":12,"title":"x","html_url":"u","state":"open"}`},
			wantErr:   "#12",
		},
		{
			name:      "a closed or merged PR is refused, naming its state",
			responses: map[string]string{"/pulls/10": `{"number":10,"title":"x","html_url":"u","state":"closed"}`, "/issues/10/comments": `[]`},
			wantErr:   "closed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeGH{responses: tc.responses, errs: tc.errs}
			got, err := fetchPRWith(context.Background(), fake.run, "o/r", testRegistry(), 10)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want an error naming %q, got %v (queue %+v)", tc.wantErr, err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("fetchPRWith: %v", err)
			}
			if len(got) != tc.wantPending {
				t.Errorf("pending = %d, want %d: %+v", len(got), tc.wantPending, got)
			}
			for _, st := range got {
				if st.PR.Number != 10 {
					t.Errorf("the scoped queue reported #%d", st.PR.Number)
				}
			}
			for _, req := range fake.seen() {
				if strings.Contains(req, wantNoList) {
					t.Errorf("the scoped queue asked for the open-PR list: %q", req)
				}
			}
		})
	}
}
