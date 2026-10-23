package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/prtriage"
)

// stubTriageFetch replaces both fetches for one test. The repo-wide one fails
// the test if reached: with --pr, the other PRs are not consulted at all.
func stubTriageFetch(t *testing.T, scoped func(n int) ([]prtriage.Status, error)) *[]int {
	t.Helper()
	var asked []int
	prevAll, prevPR := triageFetch, triageFetchPR
	triageFetch = func(context.Context, string, prtriage.Registry) ([]prtriage.Status, error) {
		t.Error("the repo-wide queue was fetched")
		return nil, nil
	}
	triageFetchPR = func(_ context.Context, _ string, _ prtriage.Registry, n int) ([]prtriage.Status, error) {
		asked = append(asked, n)
		return scoped(n)
	}
	t.Cleanup(func() { triageFetch, triageFetchPR = prevAll, prevPR })
	return &asked
}

func validRegistry(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "review-attestation.json")
	if err := os.WriteFile(path, []byte(`{"triage":{"marker":"## Review triage"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// --pr N exits on N's queue alone: pending on N is the pending error, N quiet
// is exit 0 whatever other PRs hold, and a fetch that fails is that failure,
// never an empty queue (#2549).
func TestPrTriageQueue_PRScopesTheExitStatus(t *testing.T) {
	unreadable := errors.New("gh api pull #42: HTTP 404")
	for _, tc := range []struct {
		name    string
		answer  []prtriage.Status
		err     error
		wantErr error
		wantOut string
	}{
		{name: "pending on N", answer: []prtriage.Status{{PR: prtriage.PR{Number: 42, Title: "mine"}, Pending: true, Reason: "github-actions reviewed, never triaged"}}, wantErr: errPending, wantOut: "#42"},
		{name: "N quiet", wantOut: "[OK]"},
		{name: "N unreadable", err: unreadable, wantErr: unreadable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asked := stubTriageFetch(t, func(int) ([]prtriage.Status, error) { return tc.answer, tc.err })
			stdout, _, err := execute(t, "pr", "triage-queue", "--repo", "o/r", "--registry", validRegistry(t), "--pr", "42")
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
			if !strings.Contains(stdout, tc.wantOut) {
				t.Errorf("stdout = %q, want it to contain %q", stdout, tc.wantOut)
			}
			if len(*asked) != 1 || (*asked)[0] != 42 {
				t.Errorf("scoped fetch asked for %v, want [42]", *asked)
			}
		})
	}
}

// A --pr that is not a pull request number is refused before anything is
// fetched.
func TestPrTriageQueue_PRMustBeAPositiveNumber(t *testing.T) {
	asked := stubTriageFetch(t, func(int) ([]prtriage.Status, error) { return nil, nil })
	for _, n := range []string{"0", "-3"} {
		if _, _, err := execute(t, "pr", "triage-queue", "--registry", validRegistry(t), "--pr", n); err == nil || !strings.Contains(err.Error(), "--pr") {
			t.Errorf("--pr %s: want a refusal naming --pr, got %v", n, err)
		}
	}
	if len(*asked) != 0 {
		t.Errorf("fetched %v for an invalid --pr", *asked)
	}
}

// TOOL-017 AC3: a pending PR names its findings that carry no declared class,
// and says so when they could not be read, rather than printing nothing.
func TestPrTriageQueue_NamesUnclassifiedFindings(t *testing.T) {
	pending := prtriage.Status{PR: prtriage.PR{Number: 42, Title: "mine"}, Pending: true, Reason: "github-actions reviewed, never triaged"}
	findings := pending
	findings.Unclassified = []prtriage.Comment{{Path: "a.go", Line: 3}, {Path: "b.go"}}
	unread := pending
	unread.FindingsUnread = true
	for _, tc := range []struct {
		name string
		st   prtriage.Status
		want string
	}{
		{"findings", findings, "2 finding(s) without a declared class: a.go:3, b.go"},
		{"unread", unread, "inline findings could not be read"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stubTriageFetch(t, func(int) ([]prtriage.Status, error) { return []prtriage.Status{tc.st}, nil })
			stdout, _, err := execute(t, "pr", "triage-queue", "--repo", "o/r", "--registry", validRegistry(t), "--pr", "42")
			if !errors.Is(err, errPending) {
				t.Errorf("err = %v, want errPending", err)
			}
			if !strings.Contains(stdout, tc.want) {
				t.Errorf("stdout = %q, want it to contain %q", stdout, tc.want)
			}
		})
	}
}

// A registry that names no classifying reviewer says so under a non-empty
// queue: silence there would read as "every finding is classified".
func TestPrTriageQueue_SaysWhenClassesAreNotChecked(t *testing.T) {
	st := prtriage.Status{PR: prtriage.PR{Number: 42, Title: "mine"}, Pending: true, Reason: "github-actions reviewed, never triaged"}
	stubTriageFetch(t, func(int) ([]prtriage.Status, error) { return []prtriage.Status{st}, nil })
	stdout, _, _ := execute(t, "pr", "triage-queue", "--repo", "o/r", "--registry", validRegistry(t), "--pr", "42")
	if !strings.Contains(stdout, "finding classes were not checked") {
		t.Errorf("stdout = %q, want the not-checked note", stdout)
	}
}

// A PR held for an unreproduced REAL finding names the finding and what its
// row lacks, and how to produce the missing row: the output is the only
// place a reader learns why a triaged PR is still listed.
func TestPrTriageQueue_NamesUnreproducedFindings(t *testing.T) {
	st := prtriage.Status{PR: prtriage.PR{Number: 42, Title: "mine"}, Pending: true,
		Reason: "triaged, but 1 REAL finding(s) lack the reproduction their disposition needs",
		Unreproduced: []prtriage.Unreproduced{{
			Finding: prtriage.Comment{Path: "cli/a.go", Line: 4},
			Problem: "applied without a `Reproduction: reproduced` row from `dotf pr reproduce`",
		}}}
	stubTriageFetch(t, func(int) ([]prtriage.Status, error) { return []prtriage.Status{st}, nil })
	stdout, _, err := execute(t, "pr", "triage-queue", "--repo", "o/r", "--registry", validRegistry(t), "--pr", "42")
	if !errors.Is(err, errPending) {
		t.Errorf("err = %v, want errPending", err)
	}
	for _, want := range []string{"cli/a.go:4", "applied without", "dotf pr reproduce"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout = %q, want it to contain %q", stdout, want)
		}
	}
}
