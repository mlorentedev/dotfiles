package prreview

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// The bats cases of tests/pr-agent-publish-guard.bats, ported. stubGH answers as
// the bats stub does: each `gh api` call is keyed by its subject (BASE and HEAD,
// the registry at each ref; PULL; COMMENTS), and an answer is "fail:<code>"
// (always fails), "flaky:<code>" (fails on the first call only, then answers the
// key's _OK value) or the response itself, which is already what --jq selects.

type stubGH struct {
	answers map[string]string
	calls   []string
}

func ghKey(path string) string {
	switch {
	case strings.HasSuffix(path, "contents/harness/review-attestation.json?ref=main"):
		return "BASE"
	case strings.Contains(path, "contents/harness/review-attestation.json?ref="):
		return "HEAD"
	case strings.Contains(path, "/pulls/"):
		return "PULL"
	case strings.HasSuffix(path, "/comments"):
		return "COMMENTS"
	}
	return ""
}

func (s *stubGH) run(_ context.Context, args ...string) ([]byte, []byte, error) {
	if len(args) < 2 || args[0] != "api" {
		return nil, []byte("stub gh: unexpected call"), errors.New("exit status 99")
	}
	key := ghKey(args[1])
	if key == "" {
		return nil, []byte("stub gh: unexpected call: " + strings.Join(args, " ")), errors.New("exit status 99")
	}
	s.calls = append(s.calls, key)
	answer := s.answers[key]
	fail := func(code string) ([]byte, []byte, error) {
		stderr := "gh: Gateway Timeout (HTTP " + code + ")\n"
		if code == "404" {
			stderr = "gh: Not Found (HTTP 404)\n"
		}
		return []byte(`{"message":"stub failure body"}` + "\n"), []byte(stderr), errors.New("exit status 1")
	}
	switch {
	case strings.HasPrefix(answer, "fail:"):
		return fail(strings.TrimPrefix(answer, "fail:"))
	case strings.HasPrefix(answer, "flaky:"):
		if s.count(key) == 1 {
			return fail(strings.TrimPrefix(answer, "flaky:"))
		}
		answer = s.answers[key+"_OK"]
	}
	return []byte(answer + "\n"), nil, nil
}

func (s *stubGH) count(key string) int {
	n := 0
	for _, c := range s.calls {
		if c == key {
			n++
		}
	}
	return n
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

var (
	guardRegistry = b64(`{"reviewers":[{"login":"github-actions","review_markers":["PR Reviewer Guide"]}]}`)
	guardNoEntry  = b64(`{"reviewers":[{"login":"coderabbitai[bot]","review_markers":["Walkthrough"]}]}`)
	guardReview   = `[{"user":{"login":"github-actions[bot]"},"updated_at":"2026-10-07T10:00:00Z","body":"## PR Reviewer Guide"}]`
)

// attempts is the workflow's ATTEMPTS, in execution order: Anthropic first
// (model `hk`), NaN (model `m`), Anthropic second.
func attempts(first, nan, second string) string {
	return fmt.Sprintf("%s hk\n%s m\n%s hk\n", first, nan, second)
}

type guardRun struct {
	stub *stubGH
	out  string
	err  error
	pub  string
}

// guardInputs are the bats suite's environment.
func guardInputs() GuardInputs {
	return GuardInputs{Repo: "o/r", PR: "7", BaseRef: "main", Started: "2026-10-07T09:00:00Z",
		HeadSHA: "abc123", Attempts: attempts("skipped", "success", "skipped"), RouteOutcome: "success"}
}

func guardEnv(stub *stubGH, out *strings.Builder) GuardEnv {
	return GuardEnv{Out: out, GH: stub.run, Sleep: func(time.Duration) {}}
}

func runGuard(t *testing.T, answers map[string]string, edit func(*GuardInputs)) guardRun {
	t.Helper()
	in := guardInputs()
	if edit != nil {
		edit(&in)
	}
	stub := &stubGH{answers: answers}
	var out strings.Builder
	err := Guard(context.Background(), in, guardEnv(stub, &out))
	return guardRun{stub: stub, out: out.String(), err: err}
}

func runProbe(t *testing.T, answers map[string]string) guardRun {
	t.Helper()
	stub := &stubGH{answers: answers}
	var out strings.Builder
	pub := ProbePublished(context.Background(), guardInputs(), guardEnv(stub, &out))
	return guardRun{stub: stub, out: out.String(), pub: pub}
}

func (r guardRun) has(t *testing.T, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(r.out, w) {
			t.Errorf("output lacks %q:\n%s", w, r.out)
		}
	}
}

func (r guardRun) lacks(t *testing.T, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(r.out, u) {
			t.Errorf("output has %q:\n%s", u, r.out)
		}
	}
}

func (r guardRun) passed(t *testing.T, want bool) {
	t.Helper()
	if want && r.err != nil {
		t.Fatalf("Guard = %v, want a pass:\n%s", r.err, r.out)
	}
	if !want && !errors.Is(r.err, ErrNotPublished) {
		t.Fatalf("Guard = %v, want ErrNotPublished:\n%s", r.err, r.out)
	}
}

func (r guardRun) calls(t *testing.T, key string, want int) {
	t.Helper()
	if got := r.stub.count(key); got != want {
		t.Errorf("%s calls = %d, want %d (%v)", key, got, want, r.stub.calls)
	}
}

func TestGuard_ARegistryReadThatKeepsFailingIsAnAPIFailure(t *testing.T) {
	r := runGuard(t, map[string]string{"BASE": "fail:504", "HEAD": "fail:504"}, nil)
	r.passed(t, false)
	r.has(t, "the GitHub API failed while determining whether a review marker is declared",
		"review-attestation.json?ref=main", "HTTP 504")
	r.lacks(t, "no review marker declared")
	r.calls(t, "BASE", 3)
}

func TestGuard_ARegistryReadThatFailsOnceIsRetriedCleanly(t *testing.T) {
	r := runGuard(t, map[string]string{"BASE": "flaky:502", "BASE_OK": guardRegistry, "COMMENTS": guardReview}, nil)
	r.passed(t, true)
	r.has(t, `review published (markers: ["PR Reviewer Guide"])`)
	r.calls(t, "BASE", 2)
}

func TestGuard_ARegistryWithNoEntryFailsWithTheExistingMessage(t *testing.T) {
	r := runGuard(t, map[string]string{"BASE": guardNoEntry, "HEAD": guardNoEntry}, nil)
	r.passed(t, false)
	r.has(t, "no review marker declared for github-actions")
	r.lacks(t, "GitHub API failed")
}

func TestGuard_A404IsAnAnswerAndTheHeadFallbackApplies(t *testing.T) {
	r := runGuard(t, map[string]string{"BASE": "fail:404", "HEAD": guardRegistry, "COMMENTS": guardReview}, nil)
	r.passed(t, true)
	r.has(t, "reviewer registry entry found only on the PR head")
	r.calls(t, "BASE", 1)
}

func TestGuard_AFailedHeadLookupIsAnAPIFailure(t *testing.T) {
	r := runGuard(t, map[string]string{"BASE": guardNoEntry, "PULL": "fail:503"},
		func(in *GuardInputs) { in.HeadSHA = "" })
	r.passed(t, false)
	r.has(t, "the GitHub API failed while determining whether a review marker is declared", "pulls/7")
	r.lacks(t, "no review marker declared")
	r.calls(t, "PULL", 3)
}

func TestGuard_ACommentListingThatKeepsFailingIsAnAPIFailure(t *testing.T) {
	r := runGuard(t, map[string]string{"BASE": guardRegistry, "COMMENTS": "fail:500"}, nil)
	r.passed(t, false)
	r.has(t, "the GitHub API failed while listing the PR's comments")
	r.lacks(t, "published no review")
}

func TestGuard_ACommentListingThatAnswers404IsReportedAsSuch(t *testing.T) {
	r := runGuard(t, map[string]string{"BASE": guardRegistry, "COMMENTS": "fail:404"}, nil)
	r.passed(t, false)
	r.has(t, "listing the PR's comments answered 404")
	r.calls(t, "COMMENTS", 1)
	r.lacks(t, "GitHub API failed")
}

func TestGuard_NoReviewCommentKeepsThePublishedNoReviewDiagnosis(t *testing.T) {
	r := runGuard(t, map[string]string{"BASE": guardRegistry, "COMMENTS": "[]"}, nil)
	r.passed(t, false)
	r.has(t, "PR-Agent reported success but published no review", "NaN's per-model concurrency limit")
}

func TestGuard_AnAnthropicAttemptIsNotBlamedOnNaN(t *testing.T) {
	r := runGuard(t, map[string]string{"BASE": guardRegistry, "COMMENTS": "[]"},
		func(in *GuardInputs) { in.Attempts = "success anthropic/claude-haiku-5-5\nskipped m\n" })
	r.passed(t, false)
	r.has(t, "PR-Agent reported success but published no review", "No cause is measured yet for the Anthropic attempt")
	r.lacks(t, "NaN")
}

func TestGuard_ThePublishedReviewNamesItsAttempt(t *testing.T) {
	r := runGuard(t, map[string]string{"BASE": guardRegistry, "COMMENTS": guardReview},
		func(in *GuardInputs) { in.Attempts = attempts("skipped", "failure", "success") })
	r.passed(t, true)
	r.has(t, "on hk")
}

func TestProbe(t *testing.T) {
	old := `[{"user":{"login":"github-actions[bot]"},"updated_at":"2026-10-07T08:00:00Z","body":"## PR Reviewer Guide"}]`
	pasted := `[{"user":{"login":"mallory"},"updated_at":"2026-10-07T10:00:00Z","body":"## PR Reviewer Guide"}]`
	suggestions := `[{"user":{"login":"github-actions[bot]"},"updated_at":"2026-10-07T10:00:00Z","body":"## PR Code Suggestions"}]`
	headSuggestions := b64(`{"reviewers":[{"login":"github-actions","review_markers":["PR Code Suggestions"]}]}`)
	for _, c := range []struct {
		name    string
		answers map[string]string
		want    string
		output  string
	}{
		{"a review published by this run answers true",
			map[string]string{"BASE": guardRegistry, "COMMENTS": guardReview}, "true", ""},
		{"no review answers false, the only answer that runs a second attempt",
			map[string]string{"BASE": guardRegistry, "COMMENTS": "[]"}, "false", ""},
		{"a review older than this run's start is not this run's review",
			map[string]string{"BASE": guardRegistry, "COMMENTS": old}, "false", ""},
		// Only the bot's comment is a review (#2529): anyone can paste the heading.
		{"the heading pasted by someone other than the bot answers false",
			map[string]string{"BASE": guardRegistry, "COMMENTS": pasted}, "false", ""},
		// On the head-ref fallback the marker is PR-Agent's own heading, never one
		// the head declares (#2529).
		{"on the head-ref fallback, a bot comment without PR-Agent's heading answers false",
			map[string]string{"BASE": "fail:404", "HEAD": headSuggestions, "COMMENTS": suggestions}, "false",
			"reviewer registry entry found only on the PR head"},
		{"an API that keeps failing answers unknown, never false",
			map[string]string{"BASE": guardRegistry, "COMMENTS": "fail:502"}, "unknown", "HTTP 502"},
		{"no declared marker answers unknown, never false",
			map[string]string{"BASE": guardNoEntry, "HEAD": guardNoEntry}, "unknown", ""},
		{"a comment listing that is not JSON answers unknown, never false",
			map[string]string{"BASE": guardRegistry, "COMMENTS": "<html>bad gateway</html>"}, "unknown", ""},
		{"a listing of two pages counts a review on the second",
			map[string]string{"BASE": guardRegistry, "COMMENTS": "[]\n" + guardReview}, "true", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := runProbe(t, c.answers)
			if r.pub != c.want {
				t.Errorf("ProbePublished = %q, want %q:\n%s", r.pub, c.want, r.out)
			}
			if c.output != "" {
				r.has(t, c.output)
			}
		})
	}
}

func TestGuard_TheLastAttemptThatRanIsJudged(t *testing.T) {
	for _, c := range []struct {
		name, attempts, comments, note, routeOutcome string
		pass                                         bool
		has, lacks                                   []string
	}{
		{name: "a failed second attempt is reported with its own model",
			attempts: attempts("skipped", "failure", "failure"), comments: "[]",
			has: []string{"PR-Agent ended failure on hk"}},
		{name: "Anthropic drawn first, then NaN: the NaN attempt is judged",
			attempts: attempts("failure", "cancelled", "skipped"), comments: "[]",
			has: []string{"PR-Agent ended cancelled on m"}},
		{name: "Anthropic drawn first and alone: its outcome is judged",
			attempts: attempts("success", "skipped", "skipped"), comments: guardReview, pass: true,
			has: []string{"on hk"}},
		{name: "a failure with no second attempt repeats why a member was out of the draw",
			attempts: attempts("skipped", "failure", "skipped"), note: "PR_AGENT_ANTHROPIC_API_KEY is not set",
			has: []string{"PR-Agent ended failure on m", "PR_AGENT_ANTHROPIC_API_KEY is not set"}},
		{name: "no attempt at all fails and says so",
			attempts: attempts("skipped", "skipped", "skipped"),
			has:      []string{"no review attempt ran", "no member of the review pool answered"}},
		// A draw that exits non-zero writes no outputs, so every attempt reads
		// skipped too; blaming the pool would send the reader to probe warnings
		// that were never printed (#2188 review).
		{name: "no attempt because the draw failed names the draw, not the pool",
			attempts: attempts("skipped", "skipped", "skipped"), routeOutcome: "failure",
			has: []string{"the draw itself failed"}, lacks: []string{"no member of the review pool answered"}},
		{name: "an ATTEMPTS list with nothing evaluated is no attempt, not a success",
			attempts: " hk\n m\n hk\n", has: []string{"no review attempt ran"}},
		{name: "a second attempt that reviewed after a first failure passes",
			attempts: attempts("skipped", "failure", "success"), comments: guardReview, pass: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			answers := map[string]string{"BASE": guardRegistry, "COMMENTS": c.comments}
			r := runGuard(t, answers, func(in *GuardInputs) {
				in.Attempts, in.RouteNote = c.attempts, c.note
				if c.routeOutcome != "" {
					in.RouteOutcome = c.routeOutcome
				}
			})
			r.passed(t, c.pass)
			r.has(t, c.has...)
			r.lacks(t, c.lacks...)
		})
	}
}

// The retry backoff is attempt × GUARD_RETRY_SECONDS (default 5), and a 404 is
// not retried.
func TestGuard_RetriesBackOffByTheConfiguredSeconds(t *testing.T) {
	for raw, want := range map[string][]time.Duration{
		"":  {5 * time.Second, 10 * time.Second},
		"2": {2 * time.Second, 4 * time.Second},
	} {
		stub := &stubGH{answers: map[string]string{"BASE": "fail:504", "HEAD": "fail:504"}}
		var slept []time.Duration
		in := guardInputs()
		in.RetrySeconds = raw
		var out strings.Builder
		env := GuardEnv{Out: &out, GH: stub.run, Sleep: func(d time.Duration) { slept = append(slept, d) }}
		_ = Guard(context.Background(), in, env)
		// Two reads (base, head), each three attempts with two waits.
		if len(slept) != 4 || slept[0] != want[0] || slept[1] != want[1] {
			t.Errorf("GUARD_RETRY_SECONDS=%q: slept %v, want %v twice", raw, slept, want)
		}
	}
}

// markers are printed as jq -c prints them: no HTML escaping.
func TestGuard_MarkersPrintAsJqPrintsThem(t *testing.T) {
	reg := b64(`{"reviewers":[{"login":"github-actions","review_markers":["<Guide> & co"]}]}`)
	review := `[{"user":{"login":"github-actions[bot]"},"updated_at":"2026-10-07T10:00:00Z","body":"## <Guide> & co"}]`
	r := runGuard(t, map[string]string{"BASE": reg, "COMMENTS": review}, nil)
	r.passed(t, true)
	r.has(t, `(markers: ["<Guide> & co"])`)
}
