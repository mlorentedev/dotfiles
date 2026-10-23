package prreview

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ErrNotPublished is the final guard's verdict that this run published no review.
// Why has already been written to GuardEnv.Out as ::error:: lines.
var ErrNotPublished = errors.New("pr guard: this run published no review")

// GHRunner runs `gh` with args and returns what it wrote to stdout and stderr.
// A non-nil error is a failed call; its stderr carries gh's reason.
type GHRunner func(ctx context.Context, args ...string) (stdout, stderr []byte, err error)

// GuardInputs are the environment of the script this replaces: Repo
// (GITHUB_REPOSITORY), PR (PR_NUMBER), BaseRef (BASE_REF), Started (STARTED, the
// run's start stamp), HeadSHA (HEAD_SHA, empty on issue_comment runs), and for
// the final guard Attempts (ATTEMPTS, one `<outcome> <model>` line per attempt
// in execution order), RouteNote (ROUTE_NOTE) and RouteOutcome (ROUTE_OUTCOME).
// RetrySeconds (GUARD_RETRY_SECONDS, default 5) spaces the API retries.
type GuardInputs struct {
	Repo, PR, BaseRef, Started, HeadSHA string
	Attempts, RouteNote, RouteOutcome   string
	RetrySeconds                        string
}

// GuardEnv is what the guard reaches outside itself through.
type GuardEnv struct {
	Out   io.Writer           // workflow annotations
	GH    GHRunner            // nil: the gh on PATH
	Sleep func(time.Duration) // nil: time.Sleep
}

// The ways measure ends without an answer, as the script numbers them.
const (
	measureMarkersFailed  = 2 // the API failed while reading the markers
	measureNoMarker       = 3 // no marker is declared
	measureComments404    = 4 // the comment listing answered 404
	measureCommentsFailed = 5 // the API failed while listing comments, or its answer was unreadable
)

// guard is one run: the inputs, and the API failures it has recorded so far.
type guard struct {
	in        GuardInputs
	env       GuardEnv
	apiErrors []string
	markers   []string
	found     int
}

func newGuard(in GuardInputs, env GuardEnv) *guard {
	if env.GH == nil {
		env.GH = execGH
	}
	if env.Sleep == nil {
		env.Sleep = time.Sleep
	}
	return &guard{in: in, env: env}
}

func (g *guard) say(format string, a ...any) { _, _ = fmt.Fprintf(g.env.Out, format+"\n", a...) }

// ProbePublished answers whether THIS run of the workflow has published a
// review: "true", "false" or "unknown". It sits between the first attempt and
// the second, and only a measured "false" lets the second run: a second review
// cannot duplicate a first one that does not exist. "unknown" (the API failed,
// or no marker is declared) never runs it, and is warned about.
func ProbePublished(ctx context.Context, in GuardInputs, env GuardEnv) string {
	g := newGuard(in, env)
	rc := g.measure(ctx)
	switch {
	case rc == 0 && g.found > 0:
		return "true"
	case rc == 0:
		return "false"
	}
	g.say("::warning::whether the first attempt published a review is unknown (probe status %d); "+
		"the second attempt runs only on a measured absence.", rc)
	for _, e := range g.apiErrors {
		g.say("::warning::%s", e)
	}
	return "unknown"
}

// Guard is the final guard: nil when this run published a review, otherwise
// ErrNotPublished, after naming the attempt that ran last and why.
//
// PR-Agent swallows a failed inference into a clean exit (run 32325012314: green
// job, no review, a NaN concurrency 429). ADR-032's top tier "queues or
// escalates, NEVER degrades silently", so a run that published nothing fails.
func Guard(ctx context.Context, in GuardInputs, env GuardEnv) error {
	g := newGuard(in, env)
	outcome, model := lastAttempt(in.Attempts)
	if model == "" {
		model = "its model"
	}
	switch outcome {
	case "cancelled", "failure":
		g.say("::error::PR-Agent ended %s on %s. Inspect its logs; a second attempt runs only when "+
			"no review is measured as published.", outcome, model)
		return g.fail()
	case "":
		if in.RouteOutcome == "failure" {
			g.say("::error::no review attempt ran: the draw itself failed, so no member was chosen. " +
				"Its step names why (an unknown PR_AGENT_PROVIDER, or a NaN preflight that failed on " +
				"its setup, fails it on purpose).")
		} else {
			g.say("::error::no review attempt ran: no member of the review pool answered its probe " +
				"(the preflight's and the draw's warnings name each one).")
		}
		return g.fail()
	}

	switch g.measure(ctx) {
	case measureMarkersFailed:
		// The failed call may be the registry read or the head-SHA lookup; the
		// detail lines name it, so the headline names neither.
		g.say("::error::the GitHub API failed while determining whether a review marker is declared, " +
			"so that is unknown. Re-run the job once the API answers.")
		g.sayAPIErrors()
		return ErrNotPublished
	case measureNoMarker:
		g.say("::error::no review marker declared for github-actions in harness/review-attestation.json "+
			"(checked %s and the PR head)", in.BaseRef)
		return ErrNotPublished
	case measureComments404:
		g.say("::error::listing the PR's comments answered 404: the PR, or this token's read access " +
			"to it, is gone, so no published review can be verified.")
		return ErrNotPublished
	case measureCommentsFailed:
		g.say("::error::the GitHub API failed while listing the PR's comments, so whether a review " +
			"was published is unknown. Re-run the job once the API answers.")
		g.sayAPIErrors()
		return ErrNotPublished
	}
	if g.found > 0 {
		g.say("review published (markers: %s) on %s", jqList(g.markers), model)
		return nil
	}
	g.say("::error::PR-Agent reported success but published no review (no github-actions[bot] "+
		"comment updated since %s carries any of %s).", in.Started, jqList(g.markers))
	// The measured causes are NaN's; naming them under an Anthropic attempt would
	// send whoever reads this to the wrong provider.
	if strings.HasPrefix(model, "anthropic/") {
		g.say("::error::No cause is measured yet for the Anthropic attempt. Its step log has the answer")
		g.say("::error::PR-Agent received: look for an output cut at DEFAULT_ANTHROPIC_CHAT_MAX_TOKENS")
		g.say("::error::or an error PR-Agent logged and then exited cleanly on.")
	} else {
		g.say("::error::Two causes are measured. A diff too large for a non-streamed answer: NaN's edge")
		g.say("::error::cuts one at about 125 s, and PR-Agent then exits cleanly (run 36680454975, 42K tokens;")
		g.say("::error::#1858). Or NaN's per-model concurrency limit, shared with pi, qq and hive (#1107).")
	}
	return g.fail()
}

// fail repeats why a pool member was out of the draw, then fails.
func (g *guard) fail() error {
	if g.in.RouteNote != "" {
		g.say("::error::%s", g.in.RouteNote)
	}
	return ErrNotPublished
}

func (g *guard) sayAPIErrors() {
	for _, e := range g.apiErrors {
		g.say("::error::%s", e)
	}
}

// lastAttempt is the last attempt with a known outcome. One GitHub never
// evaluated has an empty outcome, and its line must not pass its model's name
// off as one.
func lastAttempt(attempts string) (outcome, model string) {
	for _, line := range strings.Split(attempts, "\n") {
		first, rest, _ := strings.Cut(strings.TrimSpace(line), " ")
		switch first {
		case "success", "failure", "cancelled":
			outcome, model = first, strings.TrimSpace(rest)
		}
	}
	return outcome, model
}

// measure sets g.markers and g.found, and returns 0 when g.found is an answer,
// otherwise one of the measure* reasons.
//
// A review is a comment that is all three of: authored by github-actions[bot]
// (on a public repo anyone can paste the marker text), created or edited at or
// after this run's start stamp (`persistent_comment` edits the earlier Guide,
// which moves `updated_at`), and carrying a marker the reviewer registry
// declares, read at the DEFAULT branch, since a PR could redefine its own.
func (g *guard) measure(ctx context.Context) int {
	g.markers = g.readMarkers(ctx, g.in.BaseRef)
	if g.markers == nil && g.in.HeadSHA == "" {
		// issue_comment events carry no pull_request payload; ask the API for the head.
		sha, _ := g.api(ctx, "repos/"+g.in.Repo+"/pulls/"+g.in.PR, "--jq", ".head.sha")
		g.in.HeadSHA = sha
	}
	if g.markers == nil && g.in.HeadSHA != "" && g.readMarkers(ctx, g.in.HeadSHA) != nil {
		// The default branch has no github-actions entry yet: this is the PR that
		// introduces it. The head registry only proves the entry EXISTS; the
		// marker is PR-Agent's own heading, never text a PR could choose (a
		// head-supplied marker such as "#" would match any bot comment, CWE-345).
		g.markers = []string{"PR Reviewer Guide"}
		g.say("::notice::reviewer registry entry found only on the PR head; " +
			"using PR-Agent's own heading as the marker")
	}
	if g.markers == nil && len(g.apiErrors) > 0 {
		return measureMarkersFailed
	}
	if g.markers == nil {
		return measureNoMarker
	}
	path := "repos/" + g.in.Repo + "/issues/" + g.in.PR + "/comments"
	comments, ok := g.api(ctx, path, "--paginate")
	if !ok {
		// api records an outage and returns silently on a 404, so no record for
		// this call means the endpoint answered 404.
		for _, e := range g.apiErrors {
			if strings.Contains(e, "/issues/"+g.in.PR+"/comments") {
				return measureCommentsFailed
			}
		}
		return measureComments404
	}
	found, err := countReviews(comments, g.in.Started, g.markers)
	if err != nil {
		// An unreadable listing is no answer, never zero, or the probe would
		// report an absence.
		return measureCommentsFailed
	}
	g.found = found
	return 0
}

// countReviews counts the comments of a --paginate listing (one JSON array per
// page) that are reviews. updated_at is compared as a string, as jq compares
// it: both sides are GitHub's fixed-width UTC stamps.
func countReviews(listing, started string, markers []string) (int, error) {
	type comment struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		UpdatedAt string `json:"updated_at"`
		Body      string `json:"body"`
	}
	dec := json.NewDecoder(strings.NewReader(listing))
	n := 0
	for {
		var page []comment
		if err := dec.Decode(&page); err == io.EOF {
			return n, nil
		} else if err != nil {
			return 0, err
		}
		for _, c := range page {
			if c.User.Login != "github-actions[bot]" || c.UpdatedAt < started {
				continue
			}
			for _, m := range markers {
				if strings.Contains(c.Body, m) {
					n++
					break
				}
			}
		}
	}
}

// readMarkers is every marker the registry at ref declares for github-actions:
// a full review carries "## PR Reviewer Guide", an incremental one "##
// Incremental PR Reviewer Guide" (TOOL-023). It is nil when none is declared,
// or when the read failed.
func (g *guard) readMarkers(ctx context.Context, ref string) []string {
	content, ok := g.api(ctx, "repos/"+g.in.Repo+"/contents/harness/review-attestation.json?ref="+ref,
		"--jq", ".content")
	if !ok {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(content), ""))
	if err != nil {
		return nil
	}
	var reg struct {
		Reviewers []struct {
			Login         string   `json:"login"`
			ReviewMarkers []string `json:"review_markers"`
		} `json:"reviewers"`
	}
	if json.Unmarshal(raw, &reg) != nil {
		return nil
	}
	var markers []string
	for _, r := range reg.Reviewers {
		if r.Login == "github-actions" {
			markers = append(markers, r.ReviewMarkers...)
		}
	}
	return markers
}

// jqList prints markers as jq -c prints a JSON array: no HTML escaping.
func jqList(markers []string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(markers)
	return strings.TrimSuffix(buf.String(), "\n")
}

// api runs `gh api path args...` and returns its answer without the trailing
// newlines, as a shell substitution would. A call that FAILS is an outage, never
// "no marker declared" (#2069: a read timeout failed a PR that PR-Agent had
// reviewed), so a failure is retried, except a 404, which is an answer: the
// thing is absent. A call that still fails is recorded in g.apiErrors, which
// the callers report instead of a misdiagnosis. Each attempt's output is its
// own: gh prints an error body on stdout, and a failed attempt's body must not
// prefix the next one's answer.
func (g *guard) api(ctx context.Context, path string, args ...string) (string, bool) {
	backoff, err := strconv.Atoi(g.in.RetrySeconds)
	if err != nil {
		backoff = 5
	}
	var stderr []byte
	for attempt := 1; attempt <= 3; attempt++ {
		var out []byte
		out, stderr, err = g.env.GH(ctx, append([]string{"api", path}, args...)...)
		if err == nil {
			return strings.TrimRight(string(out), "\n"), true
		}
		if bytes.Contains(stderr, []byte("HTTP 404")) {
			return "", false
		}
		if attempt < 3 {
			g.env.Sleep(time.Duration(attempt*backoff) * time.Second)
		}
	}
	g.apiErrors = append(g.apiErrors, "gh api "+path+": "+strings.ReplaceAll(string(stderr), "\n", " "))
	return "", false
}

// execGH runs the gh on PATH.
func execGH(ctx context.Context, args ...string) ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
