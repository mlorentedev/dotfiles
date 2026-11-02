package prreview

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const testAuditedRef = "981344be2755e303df501e18e3f60b848a188a80"

// fakeGitHub answers Reviewable's calls. Each answer is the raw API body; "fail"
// fails the call. calls records each call's path, in order.
type fakeGitHub struct {
	pull, files          string
	config, ext, handler []byte
	failConfig, failExt  bool
	calls                []string
}

func (f *fakeGitHub) run(_ context.Context, args ...string) ([]byte, []byte, error) {
	path := args[1]
	f.calls = append(f.calls, path)
	failed := func() ([]byte, []byte, error) {
		return nil, []byte("gh: Gateway Timeout (HTTP 504)\n"), errors.New("exit status 1")
	}
	answer := func(body string) ([]byte, []byte, error) {
		if body == "fail" {
			return failed()
		}
		if rest, ok := strings.CutPrefix(body, "fail:"); ok {
			// gh prints an error's body on stdout, and exits non-zero.
			return []byte(rest), []byte("gh: Bad Gateway (HTTP 502)\n"), errors.New("exit status 1")
		}
		return []byte(body + "\n"), nil, nil
	}
	file := func(content []byte, fail bool) ([]byte, []byte, error) {
		if fail {
			return failed()
		}
		// The contents API wraps its base64 at 60 columns.
		enc := base64.StdEncoding.EncodeToString(content)
		var wrapped strings.Builder
		for len(enc) > 60 {
			wrapped.WriteString(enc[:60] + "\n")
			enc = enc[60:]
		}
		return []byte(wrapped.String() + enc + "\n"), nil, nil
	}
	switch {
	case strings.HasSuffix(path, "/pulls/7"):
		return answer(f.pull)
	case strings.HasSuffix(path, "/pulls/7/files"):
		if !slices.Contains(args, "--paginate") {
			return answer(firstPage(f.files))
		}
		return answer(f.files)
	case strings.HasPrefix(path, "repos/o/r/contents/.pr_agent.toml?ref=main"):
		return file(f.config, f.failConfig)
	case path == "repos/The-PR-Agent/pr-agent/contents/pr_agent/settings/language_extensions.toml?ref="+testAuditedRef:
		return file(f.ext, f.failExt)
	case path == "repos/The-PR-Agent/pr-agent/contents/pr_agent/algo/language_handler.py?ref="+testAuditedRef:
		return file(f.handler, false)
	}
	return nil, []byte("unexpected call " + path), errors.New("exit status 99")
}

// firstPage is what the listing answers to a call that does not paginate.
func firstPage(listing string) string {
	first, _, _ := strings.Cut(listing, "\n")
	return first
}

func newFakeGitHub(t *testing.T) *fakeGitHub {
	return &fakeGitHub{pull: openPR("feat/x"), files: `[{"filename":"cli/main.go"}]`, config: realConfig(t),
		ext: upstreamFixture(t, "language_extensions.toml"), handler: upstreamFixture(t, "language_handler.py")}
}

func openPR(branch string) string {
	return `{"merged":false,"state":"open","draft":false,"head":{"ref":"` + branch + `"}}`
}

func decide(t *testing.T, g *fakeGitHub, event, prefixes string) (bool, string, error) {
	t.Helper()
	var out bytes.Buffer
	ok, err := Reviewable(context.Background(), ReviewableInputs{Repo: "o/r", PR: "7", Event: event,
		DefaultBranch: "main", AuditedRef: testAuditedRef, SkipPrefixes: prefixes},
		ReviewableEnv{Out: &out, GH: g.run})
	return ok, out.String(), err
}

func TestReviewable_APRMergedClosedOrDraftedWhileQueuedIsSkippedAfterOneRead(t *testing.T) {
	for state, pull := range map[string]string{
		"merged": `{"merged":true,"state":"closed","draft":false}`,
		"closed": `{"merged":false,"state":"closed","draft":false}`,
		"draft":  `{"merged":false,"state":"open","draft":true}`,
	} {
		t.Run(state, func(t *testing.T) {
			g := newFakeGitHub(t)
			g.pull = pull
			ok, out, err := decide(t, g, "pull_request", "")
			if err != nil || ok {
				t.Fatalf("reviewable = %v, err %v", ok, err)
			}
			if !strings.Contains(out, "::notice::PR #7 is "+state+" now (read from the API, not the event payload)") {
				t.Errorf("notice: %s", out)
			}
			if len(g.calls) != 1 {
				t.Errorf("went past the state read: %v", g.calls)
			}
		})
	}
}

func TestReviewable_AnOpenPRGoesOnToTheEmptyDiffCheck(t *testing.T) {
	g := newFakeGitHub(t)
	ok, out, err := decide(t, g, "pull_request", "")
	if err != nil || !ok {
		t.Fatalf("reviewable = %v, err %v: %s", ok, err, out)
	}
	if out != "" {
		t.Errorf("a review needs no annotation, got %s", out)
	}
	if len(g.calls) != 5 {
		t.Errorf("calls = %v, want the PR, the config, two upstream files and the list", g.calls)
	}
}

// Only a definite answer skips: each of these leaves a file list that alone
// would skip the review, and the failed read decides it.
func TestReviewable_WhatCannotBeReadKeepsTheReview(t *testing.T) {
	ignored := `[{"filename":"sensitive/a.age"}]`
	for name, arrange := range map[string]func(*fakeGitHub){
		"the file list fails":     func(g *fakeGitHub) { g.files = "fail" },
		"the file list is empty":  func(g *fakeGitHub) { g.files = "[]" },
		"the file list is broken": func(g *fakeGitHub) { g.files = ignored + "{" },
		"the config fails":        func(g *fakeGitHub) { g.failConfig = true },
		"the config is invalid":   func(g *fakeGitHub) { g.config = []byte("[ignore\n") },
		"the extensions invalid":  func(g *fakeGitHub) { g.ext = []byte("[bad_extensions\n") },
	} {
		t.Run(name, func(t *testing.T) {
			g := newFakeGitHub(t)
			g.files = ignored
			arrange(g)
			ok, _, err := decide(t, g, "pull_request", "")
			if err != nil || !ok {
				t.Errorf("reviewable = %v, err %v, want the review to run", ok, err)
			}
		})
	}
}

func TestReviewable_AFailedStateReadDoesNotSkip(t *testing.T) {
	for name, pull := range map[string]string{
		"a failed read":                  "fail",
		"an unreadable answer":           "<html>",
		"a failed read's body on stdout": `fail:{"merged":true,"state":"closed","head":{"ref":"feat/x"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			g := newFakeGitHub(t)
			g.pull = pull
			ok, _, err := decide(t, g, "pull_request", "feat/")
			if err != nil || !ok {
				t.Errorf("reviewable = %v, err %v", ok, err)
			}
		})
	}
}

func TestReviewable_EveryFileIgnoredIsSkippedAndSaysItIsNoReview(t *testing.T) {
	g := newFakeGitHub(t)
	g.files = "[{\"filename\":\"sensitive/a.age\"}]\n[{\"filename\":\"go.sum\"}]"
	ok, out, err := decide(t, g, "pull_request", "")
	if err != nil || ok {
		t.Fatalf("reviewable = %v, err %v", ok, err)
	}
	if !strings.Contains(out, "This is not a review: review-attestation still needs CodeRabbit or a human (#1618)") {
		t.Errorf("notice: %s", out)
	}
}

func TestReviewable_EveryPageOfTheListingCounts(t *testing.T) {
	g := newFakeGitHub(t)
	g.files = "[{\"filename\":\"sensitive/a.age\"}]\n[{\"filename\":\"cli/main.go\"}]"
	if ok, _, _ := decide(t, g, "pull_request", ""); !ok {
		t.Error("a reviewable file on the second page was not counted")
	}
}

func TestReviewable_AReviewCommentReadsNeitherStateNorBranch(t *testing.T) {
	g := newFakeGitHub(t)
	g.pull = `{"merged":true,"state":"closed","draft":true,"head":{"ref":"release-please--main"}}`
	ok, _, err := decide(t, g, "issue_comment", "release-please--")
	if err != nil || !ok {
		t.Fatalf("reviewable = %v, err %v", ok, err)
	}
	for _, call := range g.calls {
		if strings.HasSuffix(call, "/pulls/7") {
			t.Errorf("a human asked, and the run read the PR's state: %v", g.calls)
		}
	}
}

func TestReviewable_AReviewCommentStillSkipsAnEmptyDiff(t *testing.T) {
	g := newFakeGitHub(t)
	g.files = `[{"filename":"sensitive/a.age"}]`
	if ok, _, err := decide(t, g, "issue_comment", ""); err != nil || ok {
		t.Errorf("reviewable = %v, err %v: PR-Agent would receive nothing", ok, err)
	}
}

func TestReviewable_ABranchPrefixForfeitsTheReview(t *testing.T) {
	cases := []struct {
		name, branch, prefixes string
		want                   bool
	}{
		{"a match skips", "release-please--branches--main", "release-please--", false},
		{"any entry of a list", "promote/x", "deploy/,promote/", false},
		{"whitespace separates too", "promote/x", "deploy/ promote/\n", false},
		{"a comma and a space", "promote/x", "deploy/, promote/", false},
		{"no match reviews", "feat/x", "deploy/,promote/", true},
		{"a prefix, not a substring", "feat/deploy/x", "deploy/", true},
		{"case matters", "Deploy/x", "deploy/", true},
		{"no prefixes reviews", "deploy/x", "", true},
		{"a trailing comma is not an empty prefix", "feat/x", "deploy/,", true},
		{"separators alone are not an empty prefix", "feat/x", " , \n", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := newFakeGitHub(t)
			g.pull = openPR(c.branch)
			ok, out, err := decide(t, g, "pull_request", c.prefixes)
			if err != nil || ok != c.want {
				t.Fatalf("reviewable = %v, err %v, want %v", ok, err, c.want)
			}
			if !c.want && len(g.calls) != 1 {
				t.Errorf("a forfeited review read more than the PR: %v", g.calls)
			}
			if !c.want && !strings.Contains(out, "forfeits the reviewer and grants nothing") {
				t.Errorf("notice: %s", out)
			}
		})
	}
}

// The ref is attacker-chosen on a public repository, so it stays out of the
// workflow's command stream.
func TestReviewable_TheBranchNameIsNotRepeated(t *testing.T) {
	g := newFakeGitHub(t)
	g.pull = openPR("deploy/secret-looking-name")
	_, out, _ := decide(t, g, "pull_request", "deploy/")
	if strings.Contains(out, "secret-looking-name") {
		t.Errorf("notice repeats the branch: %s", out)
	}
}

func TestReviewable_AnUnknownAuditedCommitFailsTheStep(t *testing.T) {
	for _, ref := range []string{"", "abc", strings.Repeat("A", 40), testAuditedRef + "0"} {
		t.Run(ref, func(t *testing.T) {
			g := newFakeGitHub(t)
			var out bytes.Buffer
			_, err := Reviewable(context.Background(), ReviewableInputs{Repo: "o/r", PR: "7",
				Event: "pull_request", DefaultBranch: "main", AuditedRef: ref},
				ReviewableEnv{Out: &out, GH: g.run})
			if !errors.Is(err, ErrNoAuditedRef) {
				t.Fatalf("err = %v", err)
			}
			if !strings.Contains(out.String(), "::error::cannot read the audited PR-Agent commit") {
				t.Errorf("annotation: %s", out.String())
			}
		})
	}
}

// A merged PR does not need the audited commit, so a broken contract does not
// turn its skip into a failure.
func TestReviewable_TheStateAnswersBeforeTheAuditedCommitIsNeeded(t *testing.T) {
	g := newFakeGitHub(t)
	g.pull = `{"merged":true,"state":"closed"}`
	ok, err := Reviewable(context.Background(), ReviewableInputs{Repo: "o/r", PR: "7", Event: "pull_request"},
		ReviewableEnv{Out: &bytes.Buffer{}, GH: g.run})
	if ok || err != nil {
		t.Errorf("reviewable = %v, err %v", ok, err)
	}
}

func TestAuditedRefFromContract(t *testing.T) {
	if got := AuditedRefFromContract(repoFile("harness", "pr-agent-upstream-contract.json")); !auditedRefPattern.MatchString(got) {
		t.Errorf("the real contract gave %q", got)
	}
	dir := t.TempDir()
	broken := filepath.Join(dir, "c.json")
	if err := os.WriteFile(broken, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{broken, filepath.Join(dir, "absent.json")} {
		if got := AuditedRefFromContract(path); got != "" {
			t.Errorf("%s gave %q", path, got)
		}
	}
}

// A branch name may forfeit a review and may never earn one: anyone who can push
// a branch can mint the name. The attestation gate's exemption is keyed on the
// diff's signature, and a name rule there would turn this forfeit into a hole.
func TestABranchNameGrantsNothing(t *testing.T) {
	for _, parts := range [][]string{
		{"scripts", "check-review-attestation.sh"},
		{"harness", "review-attestation.json"},
	} {
		raw, err := os.ReadFile(repoFile(parts...))
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"head.ref", "head_ref", "headRefName", "startsWith", "release-please--"} {
			if strings.Contains(string(raw), name) {
				t.Errorf("%s reads the branch name (%s): what grants must not", filepath.Join(parts...), name)
			}
		}
	}
}
