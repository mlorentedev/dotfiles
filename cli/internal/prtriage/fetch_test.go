package prtriage

import (
	"context"
	"errors"
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
	// longest matching substring wins, so a test can register a general list
	// ("/pulls") plus specific per-PR payloads ("/pulls/7/reviews") and get the
	// specific one every time; map order would make it a coin toss.
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
	// A reviews request is answered only by a fragment naming reviews, and
	// otherwise by "no reviews": every fixture written before #1686 describes
	// a PR with none, and "/pulls/7" must not answer for "/pulls/7/reviews".
	isReviews := strings.Contains(joined, "/reviews")
	best, found := "", false
	for frag := range f.responses {
		if isReviews && !strings.Contains(frag, "/reviews") {
			continue
		}
		if strings.Contains(joined, frag) && (!found || len(frag) > len(best)) {
			best, found = frag, true
		}
	}
	if found {
		return []byte(f.responses[best]), nil
	}
	if isReviews {
		return []byte(`[]`), nil
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

// TestFetchCarriesUpdatedAtFromTheWire is TOOL-019 (#1422) at the transport
// layer, and it is the half the domain tests cannot reach: those construct
// Comment values directly, so UpdatedAt could be absent from the REST struct or
// left unmapped in the conversion and every one of them would still pass.
//
// The fixture is the shape the defect takes in production: ONE comment created
// before the triage and edited after it. Both reviewers here re-review by
// growing an existing comment rather than posting a new one, so createdAt stays
// behind and a queue reading it reports clear over unread findings.
func TestFetchCarriesUpdatedAtFromTheWire(t *testing.T) {
	fake := &fakeGH{responses: map[string]string{
		"/pulls": `[{"number":12,"title":"a re-reviewed PR","html_url":"https://github.com/o/r/pull/12"}]`,
		"/issues/12/comments": `[{"user":{"login":"github-actions[bot]"},
		                          "body":"## PR Reviewer Guide\nfirst, then more",
		                          "created_at":"2026-09-02T10:00:00Z",
		                          "updated_at":"2026-09-02T12:00:00Z"},
		                         {"user":{"login":"manu"},
		                          "body":"## Review triage\ndispositioned",
		                          "created_at":"2026-09-02T11:00:00Z",
		                          "updated_at":"2026-09-02T11:00:00Z"}]`,
	}}

	got, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
	if err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("queue length = %d, want 1 — the reviewer edited its comment at 12:00, "+
			"after the 11:00 triage, so #12 is pending\n%+v", len(got), got)
	}
	if want := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC); !got[0].At.Equal(want) {
		t.Errorf("At = %v, want %v — REST spells this updated_at; reporting created_at here "+
			"is the whole defect, because it puts the reviewer behind the triage", got[0].At, want)
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

// The file list is read only for a PR the queue would report, so a quiet PR
// costs no extra call on the session-start path, and a release PR whose diff is
// exactly the exempt signature leaves the queue (#1196).
func TestFetchDropsAPendingPRWhoseDiffIsAnExemptSignature(t *testing.T) {
	r := testRegistry()
	r.Exempt.Signatures = []Signature{{Name: "release-please", Files: []string{"CHANGELOG.md", "versions.conf"}}}
	review := `[{"user":{"login":"coderabbitai[bot]"},"body":"No actionable comments were generated","created_at":"2026-10-09T05:00:00Z"}]`
	fake := &fakeGH{responses: map[string]string{
		"pulls?state":         `[{"number":20,"title":"release"},{"number":21,"title":"feature"},{"number":22,"title":"quiet"}]`,
		"/issues/20/comments": review,
		"/issues/21/comments": review,
		"/issues/22/comments": `[]`,
		"pulls/20/files":      `[{"filename":"versions.conf"},{"filename":"CHANGELOG.md"}]`,
		"pulls/21/files":      `[{"filename":"CHANGELOG.md"},{"filename":"cli/main.go"}]`,
	}}
	got, err := fetchWith(context.Background(), fake.run, "o/r", r)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].PR.Number != 21 {
		t.Fatalf("want only #21 pending, got %+v", got)
	}
	for _, req := range fake.seen() {
		if strings.Contains(req, "pulls/22/files") {
			t.Error("a PR with nothing pending must not cost a files call")
		}
	}
}

// A file list that fills a page cannot be compared, and is never exempt: the
// error toward a louder queue, the direction this package always takes.
func TestFetchKeepsAPRWhoseFileListFillsAPage(t *testing.T) {
	r := testRegistry()
	r.Exempt.Signatures = []Signature{{Name: "release-please", Files: []string{"CHANGELOG.md"}}}
	files := make([]string, filesLimit)
	for i := range files {
		files[i] = `{"filename":"CHANGELOG.md"}`
	}
	fake := &fakeGH{responses: map[string]string{
		"pulls?state":         `[{"number":20,"title":"release"}]`,
		"/issues/20/comments": `[{"user":{"login":"coderabbitai[bot]"},"body":"No actionable comments were generated","created_at":"2026-10-09T05:00:00Z"}]`,
		"pulls/20/files":      "[" + strings.Join(files, ",") + "]",
	}}
	got, err := fetchWith(context.Background(), fake.run, "o/r", r)
	if err != nil || len(got) != 1 {
		t.Fatalf("a full page of files must keep the PR pending: %+v, %v", got, err)
	}
}

// A files call that fails, a deadline on the session-start path among the
// causes, keeps that PR listed and does not discard the rest of the queue.
func TestFetchKeepsAPRWhoseFilesCannotBeRead(t *testing.T) {
	r := testRegistry()
	r.Exempt.Signatures = []Signature{{Name: "release-please", Files: []string{"CHANGELOG.md"}}}
	review := `[{"user":{"login":"coderabbitai[bot]"},"body":"No actionable comments were generated","created_at":"2026-10-09T05:00:00Z"}]`
	fake := &fakeGH{
		responses: map[string]string{
			"pulls?state":         `[{"number":20,"title":"release"},{"number":21,"title":"feature"}]`,
			"/issues/20/comments": review,
			"/issues/21/comments": review,
			"pulls/21/files":      `[{"filename":"cli/main.go"}]`,
		},
		errs: map[string]error{"pulls/20/files": context.DeadlineExceeded},
	}
	got, err := fetchWith(context.Background(), fake.run, "o/r", r)
	if err != nil {
		t.Fatalf("an unreadable file list must not fail the queue: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("both PRs stay listed, got %+v", got)
	}
}

// TestFetchCountsAFormalReviewFiledAfterTheTriage is #1686, measured twice: on
// #1683 and on #2581, which merged over it. CodeRabbit files a review WITH
// findings through the reviews API, with inline comments and no marker, so a
// queue that reads only issue comments never sees it and answers clear. The
// timestamps are #2581's: triage at 03:04:46, review at 03:19:14, merge 82
// seconds later.
func TestFetchCountsAFormalReviewFiledAfterTheTriage(t *testing.T) {
	fake := &fakeGH{responses: map[string]string{
		"/pulls": `[{"number":2581,"title":"t","html_url":"u"}]`,
		"/issues/2581/comments": `[{"user":{"login":"github-actions[bot]"},"body":"## PR Reviewer Guide\nf",
		                            "created_at":"2026-10-16T02:55:00Z"},
		                           {"user":{"login":"mlorentedev"},"body":"## Review triage\nall declined",
		                            "created_at":"2026-10-16T03:04:46Z"}]`,
		"/pulls/2581/reviews": `[{"user":{"login":"coderabbitai[bot]"},"state":"COMMENTED",
		                          "body":"**Actionable comments posted: 3**","submitted_at":"2026-10-16T03:19:14Z"}]`,
	}}

	got, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
	if err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("queue length = %d, want 1: a review filed after the triage is output the triage did not read", len(got))
	}
	if got[0].Reviewer != "coderabbitai" || got[0].Reason != "coderabbitai reviewed again after the last triage" {
		t.Errorf("Reviewer = %q, Reason = %q", got[0].Reviewer, got[0].Reason)
	}
	var asked bool
	for _, req := range fake.seen() {
		asked = asked || strings.Contains(req, "/pulls/2581/reviews")
	}
	if !asked {
		t.Error("the reviews endpoint was never called")
	}
}

// TestFetchIgnoresDismissedAndPendingReviews: a dismissed review was withdrawn
// and a pending one was never submitted; neither is output owed a disposition.
func TestFetchIgnoresDismissedAndPendingReviews(t *testing.T) {
	fake := &fakeGH{responses: map[string]string{
		"/pulls":             `[{"number":5,"title":"t","html_url":"u"}]`,
		"/issues/5/comments": `[]`,
		"/pulls/5/reviews": `[{"user":{"login":"coderabbitai[bot]"},"state":"DISMISSED","body":"x","submitted_at":"2026-10-16T03:00:00Z"},
		                      {"user":{"login":"coderabbitai[bot]"},"state":"PENDING","body":"y"}]`,
	}}

	got, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
	if err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("queue = %+v, want empty", got)
	}
}

// TestFetchReviewTriagedAfterwardsIsClear: the reverse order of #2581. A
// triage written after the review read it, so the queue is clear.
func TestFetchReviewTriagedAfterwardsIsClear(t *testing.T) {
	fake := &fakeGH{responses: map[string]string{
		"/pulls": `[{"number":8,"title":"t","html_url":"u"}]`,
		"/issues/8/comments": `[{"user":{"login":"mlorentedev"},"body":"## Review triage\napplied",
		                         "created_at":"2026-10-16T03:30:00Z"}]`,
		"/pulls/8/reviews": `[{"user":{"login":"coderabbitai[bot]"},"state":"COMMENTED","body":"3 findings",
		                       "submitted_at":"2026-10-16T03:19:14Z"}]`,
	}}

	got, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
	if err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("queue = %+v, want empty: the triage is newer than the review", got)
	}
}

// TestFetchRefusesAFullPageOfReviews: the reviews axis truncates the same way
// the comments axis does, and the newest review is the one a short page drops.
func TestFetchRefusesAFullPageOfReviews(t *testing.T) {
	items := make([]string, commentLimit)
	for i := range items {
		items[i] = `{"user":{"login":"someone"},"state":"COMMENTED","body":"b","submitted_at":"2026-10-16T03:00:00Z"}`
	}
	fake := &fakeGH{responses: map[string]string{
		"/pulls":             `[{"number":9,"title":"t","html_url":"u"}]`,
		"/issues/9/comments": `[]`,
		"/pulls/9/reviews":   "[" + strings.Join(items, ",") + "]",
	}}

	_, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
	if err == nil || !strings.Contains(err.Error(), "full page") {
		t.Fatalf("err = %v, want a full-page refusal", err)
	}
}

// TestFetchDoesNotCountANoticeAsReview is #1686's second criterion: a
// rate-limit or in-progress notice is status, not review. It carries no
// marker, and it is an issue comment, so reading reviews must not change that.
func TestFetchDoesNotCountANoticeAsReview(t *testing.T) {
	fake := &fakeGH{responses: map[string]string{
		"/pulls": `[{"number":6,"title":"t","html_url":"u"}]`,
		"/issues/6/comments": `[{"user":{"login":"coderabbitai[bot]"},
		                         "body":"<!-- rate limited by coderabbit.ai -->\n## Review limit reached",
		                         "created_at":"2026-10-16T03:25:58Z"}]`,
		"/pulls/6/reviews": `[]`,
	}}

	got, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry())
	if err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("queue = %+v, want empty: a notice is not review output", got)
	}
}

// TestFetchFailsWhenReviewsCannotBeRead is #1686's fourth criterion: a PR
// whose reviews could not be read is not a PR with no reviews.
func TestFetchFailsWhenReviewsCannotBeRead(t *testing.T) {
	fake := &fakeGH{
		responses: map[string]string{
			"/pulls":             `[{"number":4,"title":"t","html_url":"u"}]`,
			"/issues/4/comments": `[]`,
		},
		errs: map[string]error{"/pulls/4/reviews": fmt.Errorf("HTTP 502")},
	}

	if _, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry()); err == nil {
		t.Fatal("fetchWith returned a queue; an unreadable reviews endpoint must fail it")
	}
}

// classifyingRegistry asks github-actions for a declared class (TOOL-017 AC3).
func classifyingRegistry() Registry {
	r := testRegistry()
	r.Classification.Prefix = "Class: "
	r.Classification.Classes = []string{"REAL", "THEORETICAL", "SPECULATIVE"}
	r.Classification.Reviewers = []string{"github-actions"}
	return r
}

var reviewedPR10 = map[string]string{
	"/pulls": `[{"number":10,"title":"a reviewed PR","html_url":"https://github.com/o/r/pull/10"}]`,
	"/issues/10/comments": `[{"user":{"login":"github-actions[bot]"},
	                          "body":"## PR Reviewer Guide\nfindings here",
	                          "created_at":"2026-09-02T10:00:00Z"}]`,
}

// The inline findings live on pulls/N/comments, which the verdict never needed
// and so never read. A declared class is only checkable there. Replies are not
// findings, so they are skipped.
func TestFetchReportsUnclassifiedInlineFindings(t *testing.T) {
	responses := map[string]string{
		"/pulls/10/comments": `[
		  {"id":1,"user":{"login":"github-actions[bot]"},"body":"**A**\n\nClass: REAL\nx","path":"a.go","line":4,"created_at":"2026-09-02T10:00:01Z"},
		  {"id":2,"user":{"login":"github-actions[bot]"},"body":"**B**\n\nREAL: prose","path":"b.go","line":9,"created_at":"2026-09-02T10:00:01Z"},
		  {"id":3,"in_reply_to_id":2,"user":{"login":"github-actions[bot]"},"body":"a reply","path":"b.go","line":9,"created_at":"2026-09-02T10:05:00Z"}]`,
	}
	for k, v := range reviewedPR10 {
		responses[k] = v
	}
	fake := &fakeGH{responses: responses}
	got, err := fetchWith(context.Background(), fake.run, "o/r", classifyingRegistry())
	if err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("queue length = %d, want 1", len(got))
	}
	u := got[0].Unclassified
	if len(u) != 1 || u[0].Path != "b.go" || u[0].Line != 9 {
		t.Fatalf("Unclassified = %+v, want exactly b.go:9", u)
	}
	if got[0].FindingsUnread {
		t.Error("FindingsUnread set although the inline comments were read")
	}
}

// A registry that asks no reviewer for a class costs no request: the
// session-start probe runs under a deadline.
func TestFetchReadsNoInlineFindingsWithoutADeclaration(t *testing.T) {
	fake := &fakeGH{responses: reviewedPR10}
	if _, err := fetchWith(context.Background(), fake.run, "o/r", testRegistry()); err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	for _, req := range fake.seen() {
		if strings.Contains(req, "/pulls/10/comments") {
			t.Fatalf("read inline comments with no classifying reviewer declared: %q", req)
		}
	}
}

// Findings that cannot be read are named as unread, never reported as none.
// The PR stays listed: the classes are a report, and losing them must not
// drop the verdict.
func TestFetchMarksUnreadInlineFindings(t *testing.T) {
	fake := &fakeGH{responses: reviewedPR10, errs: map[string]error{"/pulls/10/comments": errors.New("boom")}}
	got, err := fetchWith(context.Background(), fake.run, "o/r", classifyingRegistry())
	if err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	if len(got) != 1 || !got[0].FindingsUnread || got[0].Unclassified != nil {
		t.Fatalf("got %+v, want PR #10 listed with FindingsUnread and no findings", got)
	}
}

// A full page of inline comments cannot be told apart from a truncated one, so
// its findings are unread, never "none unclassified". Same rule as the
// conversation and the reviews, which refuse a full page outright.
func TestFetchMarksAFullPageOfInlineFindingsUnread(t *testing.T) {
	var page strings.Builder
	page.WriteString("[")
	for i := range commentLimit {
		if i > 0 {
			page.WriteString(",")
		}
		fmt.Fprintf(&page, `{"id":%d,"user":{"login":"github-actions[bot]"},"body":"Class: REAL","path":"a.go","line":1,"created_at":"2026-09-02T10:00:01Z"}`, i+1)
	}
	page.WriteString("]")
	responses := map[string]string{"/pulls/10/comments": page.String()}
	for k, v := range reviewedPR10 {
		responses[k] = v
	}
	got, err := fetchWith(context.Background(), (&fakeGH{responses: responses}).run, "o/r", classifyingRegistry())
	if err != nil {
		t.Fatalf("fetchWith: %v", err)
	}
	if len(got) != 1 || !got[0].FindingsUnread {
		t.Fatalf("got %+v, want PR #10 listed with FindingsUnread", got)
	}
}

// The registry key is spelled `reviewers` under `classification`. A typo in
// either side would leave the report silently inert with every other test
// green, so the JSON spelling is pinned here.
func TestParseRegistryReadsTheClassifyingReviewers(t *testing.T) {
	r, err := ParseRegistry([]byte(`{"triage":{"marker":"## Review triage"},
	  "classification":{"prefix":"Class: ","classes":["REAL"],"reviewers":["github-actions"]}}`), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Classification.Reviewers) != 1 || r.Classification.Reviewers[0] != "github-actions" {
		t.Fatalf("Classification.Reviewers = %v, want [github-actions]", r.Classification.Reviewers)
	}
}
