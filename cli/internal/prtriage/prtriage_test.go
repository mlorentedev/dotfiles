package prtriage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func at(min int) time.Time {
	return time.Date(2026, 8, 18, 1, min, 0, 0, time.UTC)
}

func reg() Registry {
	var r Registry
	r.Reviewers = []Reviewer{
		{Login: "coderabbitai", ReviewMarkers: nil},
		{Login: "github-actions", ReviewMarkers: []string{"## PR Reviewer Guide"}},
	}
	r.Triage.Marker = "## Review triage"
	return r
}

// AC7/AC8: the queue's whole contract, as a table. Each row is a state the loop
// actually reaches, not a synthetic permutation.
func TestEvaluate(t *testing.T) {
	cases := []struct {
		name     string
		comments []Comment
		pending  bool
	}{
		{
			name:     "no comments at all",
			comments: nil,
			pending:  false,
		},
		{
			name: "a reviewer spoke and nobody triaged",
			comments: []Comment{
				{Author: "github-actions", Body: "## PR Reviewer Guide\nfindings", CreatedAt: at(10)},
			},
			pending: true,
		},
		{
			name: "triaged after the review",
			comments: []Comment{
				{Author: "github-actions", Body: "## PR Reviewer Guide\nfindings", CreatedAt: at(10)},
				{Author: "manu", Body: "## Review triage\n| item | applied |", CreatedAt: at(20)},
			},
			pending: false,
		},
		{
			// The case that matters. Pushing a fix makes the reviewer re-review,
			// and the earlier disposition does not cover what it said this time.
			name: "the reviewer spoke again after the triage",
			comments: []Comment{
				{Author: "github-actions", Body: "## PR Reviewer Guide\nfirst", CreatedAt: at(10)},
				{Author: "manu", Body: "## Review triage\ndone", CreatedAt: at(20)},
				{Author: "github-actions", Body: "## PR Reviewer Guide\nsecond", CreatedAt: at(30)},
			},
			pending: true,
		},
		{
			// GraphQL says `github-actions`, REST says `github-actions[bot]`.
			// Matching raw made the attestation gate depend on which API produced
			// the payload (#1033); the queue must not inherit that.
			name: "the REST spelling of the same reviewer",
			comments: []Comment{
				{Author: "github-actions[bot]", Body: "## PR Reviewer Guide\nx", CreatedAt: at(10)},
			},
			pending: true,
		},
		{
			// A reviewer with no markers publishes proper reviews; its ordinary
			// comments are not review output and must not enqueue anything.
			name: "a declared reviewer commenting without a marker",
			comments: []Comment{
				{Author: "coderabbitai", Body: "auto-summary, not a review", CreatedAt: at(10)},
			},
			pending: false,
		},
		{
			name: "an undeclared author using the marker verbatim",
			comments: []Comment{
				{Author: "somebody-else", Body: "## PR Reviewer Guide\nnot a declared reviewer", CreatedAt: at(10)},
			},
			pending: false,
		},
		{
			// The marker is matched at the start of a line, so quoting it in
			// prose — as this repo's own docs do — is not a disposition.
			name: "the triage marker quoted mid-sentence does not count",
			comments: []Comment{
				{Author: "github-actions", Body: "## PR Reviewer Guide\nx", CreatedAt: at(10)},
				{Author: "manu", Body: "we should write a ## Review triage section soon", CreatedAt: at(20)},
			},
			pending: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(PR{Number: 1, Comments: tc.comments}, reg())
			if got.Pending != tc.pending {
				t.Fatalf("pending = %v, want %v (reason: %s)", got.Pending, tc.pending, got.Reason)
			}
			if got.Reason == "" {
				t.Fatal("Reason is empty: a verdict that cannot say why is the defect this repo keeps finding")
			}
		})
	}
}

// A registry with no triage marker must fail rather than silently match every
// comment, which would empty the queue and look like "nothing pending".
func TestLoadRegistryRefusesAnEmptyMarker(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "reg.json")
	if err := os.WriteFile(p, []byte(`{"reviewers":[],"triage":{"marker":"  "}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRegistry(p); err == nil {
		t.Fatal("want an error for an empty triage.marker, got nil")
	}
}

// AC9: the queue and the gate read the SAME file. This asserts the real one
// parses here and carries what both consumers need — a marker, and at least one
// reviewer whose output is comment-shaped. Without it the two could drift into
// separate truths, which is the failure class the registry exists to prevent.
func TestTheRealRegistryServesBothConsumers(t *testing.T) {
	r, err := LoadRegistry(filepath.Join("..", "..", "..", "harness", "review-attestation.json"))
	if err != nil {
		t.Fatalf("the shipped registry does not load: %v", err)
	}
	if len(r.Reviewers) == 0 {
		t.Fatal("the shipped registry declares no reviewers")
	}
	markers := 0
	for _, rv := range r.Reviewers {
		markers += len(rv.ReviewMarkers)
	}
	if markers == 0 {
		t.Fatal("no reviewer declares review_markers: the queue can never see comment-shaped output")
	}
}

// 100 open PRs indicates boundary truncation on `gh pr list --limit 100`.
// It must return a loud error rather than silently returning a truncated queue.
func TestParseWireBoundaryLimit(t *testing.T) {
	r := reg()

	// Under 100: success
	out99 := `[{"number": 1, "title": "t", "url": "u", "comments": [{"author": {"login": "github-actions"}, "body": "## PR Reviewer Guide\nx", "createdAt": "2026-08-20T01:00:00Z"}]}]`
	got, err := parseWire([]byte(out99), r)
	if err != nil {
		t.Fatalf("parseWire with 1 PR returned unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}

	// 100 entries: boundary limit error
	var sb strings.Builder
	sb.WriteString("[")
	for i := 1; i <= 100; i++ {
		if i > 1 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"number": %d, "title": "t%d", "url": "u%d", "comments": []}`, i, i, i)
	}
	sb.WriteString("]")

	_, err = parseWire([]byte(sb.String()), r)
	if err == nil {
		t.Fatal("expected loud error when wire returns 100 items (boundary limit), got nil")
	}
	if !strings.Contains(err.Error(), "hit boundary limit") || !strings.Contains(err.Error(), "100") {
		t.Fatalf("error must cite boundary limit and count, got: %v", err)
	}
}
