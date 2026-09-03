package prtriage

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeGH stands in for the `gh` binary. It exists because the fetch path had no
// seam at all before CLI-071: `FetchWithRegistry` reached for exec.CommandContext
// directly, so the one code path observed failing in production (#1454) was the
// one path no test could reach.
//
// It records every request, so a test can assert which endpoints were called and
// not merely what came back — the difference between proving the transport and
// hoping about it.
type fakeGH struct {
	mu        sync.Mutex
	requests  []string
	inFlight  int
	maxFlight int

	// responses maps a substring of the request path to its canned body. The
	// first match wins, so a test can register a general list plus specific
	// per-PR comment payloads.
	responses map[string]string
	errs      map[string]error
	delay     time.Duration
}

func (f *fakeGH) run(ctx context.Context, args ...string) ([]byte, error) {
	joined := strings.Join(args, " ")

	f.mu.Lock()
	f.requests = append(f.requests, joined)
	f.inFlight++
	if f.inFlight > f.maxFlight {
		f.maxFlight = f.inFlight
	}
	f.mu.Unlock()

	defer func() {
		f.mu.Lock()
		f.inFlight--
		f.mu.Unlock()
	}()

	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	for frag, err := range f.errs {
		if strings.Contains(joined, frag) {
			return nil, err
		}
	}
	for frag, body := range f.responses {
		if strings.Contains(joined, frag) {
			return []byte(body), nil
		}
	}
	return nil, fmt.Errorf("fakeGH: no canned response for %q", joined)
}

func (f *fakeGH) seen() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.requests))
	copy(out, f.requests)
	return out
}

// testRegistry mirrors the real harness/review-attestation.json: reviewers are
// declared with BARE logins, which is the GraphQL spelling. REST returns the
// `[bot]` suffix, so the fold in normaliseLogin is what makes these agree.
func testRegistry() Registry {
	var r Registry
	r.Triage.Marker = "## Review triage"
	r.Reviewers = []Reviewer{
		{Login: "coderabbitai", ReviewMarkers: []string{"No actionable comments were generated"}},
		{Login: "github-actions", ReviewMarkers: []string{"## PR Reviewer Guide"}},
	}
	return r
}

// TestFetchWithRunnerDrivesTheWholePath is AC1: the fetch path runs end to end
// with no network and no `gh` on PATH. Before CLI-071 this test could not be
// written at all.
func TestFetchWithRunnerDrivesTheWholePath(t *testing.T) {
	fake := &fakeGH{responses: map[string]string{
		"/pulls": `[{"number":10,"title":"a reviewed PR","html_url":"https://github.com/o/r/pull/10"},
		            {"number":11,"title":"a quiet PR","html_url":"https://github.com/o/r/pull/11"}]`,
		"/issues/10/comments": `[{"user":{"login":"github-actions[bot]"},
		                          "body":"## PR Reviewer Guide\nfindings here",
		                          "created_at":"2026-09-02T10:00:00Z"}]`,
		"/issues/11/comments": `[]`,
	}}

	got, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
	if err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("queue length = %d, want 1 (only #10 carries untriaged reviewer output)\n%+v", len(got), got)
	}
	// AC6 rides here: every field the two renderers consume is pinned, so a
	// transport that parsed but mismapped — REST spells three of these
	// differently from the GraphQL shape it replaced — cannot pass by producing
	// a queue of the right length with hollow entries.
	st := got[0]
	if st.PR.Number != 10 {
		t.Errorf("Number = %d, want 10", st.PR.Number)
	}
	if st.PR.Title != "a reviewed PR" {
		t.Errorf("Title = %q, want %q", st.PR.Title, "a reviewed PR")
	}
	if st.PR.URL != "https://github.com/o/r/pull/10" {
		t.Errorf("URL = %q — REST calls this field html_url, not url", st.PR.URL)
	}
	if !st.Pending {
		t.Errorf("PR #10 has reviewer output and no triage record; want Pending")
	}
	if st.Reviewer != "github-actions" {
		t.Errorf("Reviewer = %q, want the registry's bare login %q", st.Reviewer, "github-actions")
	}
	if want := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC); !st.At.Equal(want) {
		t.Errorf("At = %v, want %v — REST spells this created_at, not createdAt, and a zero "+
			"timestamp silently turns the queue's comparison into a wrong answer", st.At, want)
	}
	if st.Reason != "github-actions reviewed, never triaged" {
		t.Errorf("Reason = %q", st.Reason)
	}
}

// TestFetchUsesRESTOnly is AC2. The point of the change is that no GraphQL call
// remains, and the only way to keep that true is to assert on the requests
// themselves — a reintroduced `gh pr list` would otherwise pass every other test
// in this file, since the queue it produces is identical.
func TestFetchUsesRESTOnly(t *testing.T) {
	fake := &fakeGH{responses: map[string]string{
		"/pulls":             `[{"number":7,"title":"t","html_url":"u"}]`,
		"/issues/7/comments": `[]`,
	}}

	if _, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry()); err != nil {
		t.Fatalf("fetchWith: %v", err)
	}

	seen := fake.seen()
	if len(seen) == 0 {
		t.Fatal("no requests recorded")
	}
	for _, req := range seen {
		if !strings.HasPrefix(req, "api ") {
			t.Errorf("request %q is not a `gh api` REST call — GraphQL is what CLI-071 removed", req)
		}
		if strings.Contains(req, "pr list") || strings.Contains(req, "graphql") {
			t.Errorf("request %q reintroduces the GraphQL path", req)
		}
	}
}

// TestFetchFoldsTheBotLoginSuffix is AC3, and it is the highest-consequence test
// here. The registry declares `github-actions`; REST returns
// `github-actions[bot]`. Under GraphQL normaliseLogin was defensive. Under REST
// it is load-bearing on every single match, and deleting it does not error — it
// empties the queue, which is the precise failure this package exists to prevent
// (#1033).
func TestFetchFoldsTheBotLoginSuffix(t *testing.T) {
	for _, login := range []string{"coderabbitai[bot]", "github-actions[bot]"} {
		t.Run(login, func(t *testing.T) {
			marker := "No actionable comments were generated"
			if strings.HasPrefix(login, "github-actions") {
				marker = "## PR Reviewer Guide"
			}
			fake := &fakeGH{responses: map[string]string{
				"/pulls": `[{"number":3,"title":"t","html_url":"u"}]`,
				"/issues/3/comments": fmt.Sprintf(
					`[{"user":{"login":%q},"body":%q,"created_at":"2026-09-02T10:00:00Z"}]`,
					login, marker+"\nbody"),
			}}

			got, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
			if err != nil {
				t.Fatalf("fetchWith: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("queue is empty for author %q — the [bot] suffix is not being folded", login)
			}
		})
	}
}

// TestFetchRefusesTruncationOnBothAxes is AC4. REST paginates where GraphQL did
// not, so there are now two ways to silently answer from partial data. The
// comment axis is the dangerous one: a truncated PR list makes a PR go missing
// (visible), while truncated comments make a triaged PR look untriaged, or worse,
// an untriaged one look clean.
func TestFetchRefusesTruncationOnBothAxes(t *testing.T) {
	t.Run("a full page of pull requests", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("[")
		for i := 0; i < prLimit; i++ {
			if i > 0 {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, `{"number":%d,"title":"t","html_url":"u"}`, i+1)
		}
		sb.WriteString("]")

		fake := &fakeGH{responses: map[string]string{
			"/pulls":    sb.String(),
			"/comments": `[]`,
		}}
		_, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
		if err == nil {
			t.Fatal("a full page of pull requests must refuse, not answer from partial data")
		}
		if !strings.Contains(err.Error(), "truncated") {
			t.Errorf("error %q should say the queue may be truncated", err)
		}
	})

	t.Run("a full page of comments on one PR", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("[")
		for i := 0; i < commentLimit; i++ {
			if i > 0 {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, `{"user":{"login":"someone"},"body":"c%d","created_at":"2026-09-02T10:00:00Z"}`, i)
		}
		sb.WriteString("]")

		fake := &fakeGH{responses: map[string]string{
			"/pulls":             `[{"number":5,"title":"t","html_url":"u"}]`,
			"/issues/5/comments": sb.String(),
		}}
		_, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
		if err == nil {
			t.Fatal("a full page of comments must refuse: a truncated comment list yields a WRONG verdict, not a missing one")
		}
		if !strings.Contains(err.Error(), "#5") {
			t.Errorf("error %q should name the pull request whose comments were truncated", err)
		}
	})
}

// TestFetchFansOutConcurrently is AC5, the session-start latency budget
// (mem.go bounds the probe at 5s and it runs on every session start). Asserting
// observed concurrency rather than wall-clock keeps this deterministic — a
// timing assertion would be flaky on a loaded CI runner and would eventually be
// deleted, which is how a guard becomes decorative.
func TestFetchFansOutConcurrently(t *testing.T) {
	const prs = 10
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < prs; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"number":%d,"title":"t","html_url":"u"}`, i+1)
	}
	sb.WriteString("]")

	fake := &fakeGH{
		delay: 20 * time.Millisecond,
		responses: map[string]string{
			"/pulls":    sb.String(),
			"/comments": `[]`,
		},
	}

	start := time.Now()
	if _, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry()); err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	elapsed := time.Since(start)

	fake.mu.Lock()
	peak := fake.maxFlight
	fake.mu.Unlock()

	if peak < 2 {
		t.Errorf("peak in-flight requests = %d: the per-PR calls are serial, so %d open PRs cost %d round-trips against a 5s session-start budget",
			peak, prs, prs+1)
	}
	if peak > commentFanout {
		t.Errorf("peak in-flight requests = %d, exceeds the %d cap: unbounded fan-out is its own failure mode", peak, commentFanout)
	}
	if serial := prs * fake.delay; elapsed > serial {
		t.Errorf("elapsed %v is no better than issuing all %d calls serially (%v)", elapsed, prs, serial)
	}
}

// TestFetchReportsFailureRatherThanAnEmptyQueue pins the rule the package
// documents: a queue that could not be computed must never read as a queue with
// nothing in it. Both axes can fail, so both are checked.
func TestFetchReportsFailureRatherThanAnEmptyQueue(t *testing.T) {
	t.Run("the list call fails", func(t *testing.T) {
		fake := &fakeGH{errs: map[string]error{"/pulls": fmt.Errorf("gh: exit status 1")}}
		got, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
		if err == nil {
			t.Fatalf("want an error, got a queue of %d", len(got))
		}
	})

	t.Run("one PR's comments fail", func(t *testing.T) {
		fake := &fakeGH{
			responses: map[string]string{"/pulls": `[{"number":9,"title":"t","html_url":"u"}]`},
			errs:      map[string]error{"/issues/9/comments": fmt.Errorf("gh: exit status 1")},
		}
		got, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
		if err == nil {
			t.Fatalf("want an error, got a queue of %d — a PR whose comments could not be read is not a PR with no comments", len(got))
		}
	})
}

// TestFetchTargetsTheCurrentRepoByDefault pins the empty-repo default. gh
// substitutes {owner}/{repo} from the current repository, which is how the old
// `gh pr list` default behaved and what the session-start probe relies on: it
// passes "" and runs wherever the shell happens to be.
func TestFetchTargetsTheCurrentRepoByDefault(t *testing.T) {
	fake := &fakeGH{responses: map[string]string{"/pulls": `[]`}}
	if _, err := fetchWith(context.Background(), fake.run, "", testRegistry()); err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	seen := fake.seen()
	if len(seen) != 1 {
		t.Fatalf("want exactly one request for an empty PR list, got %d: %v", len(seen), seen)
	}
	if !strings.Contains(seen[0], "repos/{owner}/{repo}/pulls") {
		t.Errorf("request %q should use gh's {owner}/{repo} placeholders when no repo is given", seen[0])
	}
}
