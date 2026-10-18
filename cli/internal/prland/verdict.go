package prland

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// A review-attestation verdict reaches a PR minutes after the review does: the
// workflow that computes it waits for a runner first (#2724). Measured on
// 2026-10-17, a workflow_run verdict started a median 203 s after it was
// created in a busy window and 457 s at most; on #2718 the status stayed at an
// older, correct "fail" for 4 min 55 s after pr-agent's Guide was posted, and
// `pr land` read it there. So land waits for a verdict it has reason to expect
// before it refuses on a red one, and says what it waited for.
const (
	// attestationCheck is the commit status review-attestation.yml publishes.
	attestationCheck = "review-attestation"
	// reviewerWorkflow is the pr-agent workflow whose run on the head, still
	// queued or running, means a review (and the verdict after it) is coming.
	reviewerWorkflow = "pr-agent.yml"
	// DefaultVerdictWait outlasts the longest verdict measured, 457 s. Past it
	// the verdict is not late, it is not coming, and the refusal says so.
	DefaultVerdictWait = 10 * time.Minute
	verdictPause       = 30 * time.Second
)

// verdictWait is what waiting for the verdict came to.
type verdictWait struct {
	For        time.Duration
	What       string // what the last round waited for
	Unreadable error  // why the signals could not be read; nothing was waited for
}

// reasons are what a refusal adds to the red check's own reason, and only
// while the check is still red: a verdict that turned green needs no note.
func (w verdictWait) reasons(f Facts) []string {
	switch {
	case !attestationFailed(f.Checks):
		return nil
	case w.Unreadable != nil:
		return []string{fmt.Sprintf("could not check whether a %s verdict is in flight: %v", attestationCheck, w.Unreadable)}
	case w.For > 0:
		return []string{fmt.Sprintf("waited %s for the %s verdict (%s) and it is still red", w.For, attestationCheck, w.What)}
	}
	return nil
}

func attestationFailed(cs []Check) bool {
	for _, c := range cs {
		if c.Name == attestationCheck && c.Bucket == "fail" {
			return true
		}
	}
	return false
}

// awaitVerdict re-reads the facts while the attestation check is red and a
// verdict is still on its way, for at most the budget. A red verdict with
// nothing coming returns at once: the refusal is then the truth. Only a
// failed read of the signals or of the facts is an error or a note; neither is
// ever read as "nothing in flight".
func awaitVerdict(ctx context.Context, gh ghFunc, o Options, number int, f Facts) (Facts, verdictWait, error) {
	var w verdictWait
	budget := o.VerdictWait
	switch {
	case budget < 0:
		return f, w, nil
	case budget == 0:
		budget = DefaultVerdictWait
	}
	sleep := o.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	for w.For < budget && attestationFailed(f.Checks) && ctx.Err() == nil {
		what, err := verdictInFlight(ctx, gh, o, number, f)
		if err != nil {
			w.Unreadable = err
			break
		}
		if what == "" {
			break
		}
		if o.Notify != nil {
			o.Notify(fmt.Sprintf("#%d: waiting for the %s verdict: %s", number, attestationCheck, what))
		}
		pause := min(verdictPause, budget-w.For)
		sleep(pause)
		w.For += pause
		w.What = what
		unknownFor := f.UnknownFor
		var err2 error
		if f, err2 = readFacts(ctx, gh, o, number); err2 != nil {
			return f, w, err2
		}
		f.UnknownFor = unknownFor
	}
	return f, w, nil
}

// verdictInFlight says what makes a verdict likely to arrive, or "" when
// nothing does. Two signals, because the first is not enough: the attestation
// run that follows pr-agent is a workflow_run, and the Actions API ties it to
// the base branch's commit, not to the PR head (#2724), so in the gap that
// failed #2718 no run on the head was active at all.
//
//  1. a pr-agent run on the head is queued or running: the review, and the
//     verdict after it, have not happened yet;
//  2. the newest comment or review on the PR is newer than the status: every
//     comment, and every pr-agent completion, makes the workflow recompute and
//     rewrite the status, so a status older than the last word on the PR has
//     not caught up with it.
func verdictInFlight(ctx context.Context, gh ghFunc, o Options, number int, f Facts) (string, error) {
	out, err := gh("run", "list", "--workflow", reviewerWorkflow, "--commit", f.HeadSHA, "--json", "status", "--limit", "20")
	if err != nil {
		return "", fmt.Errorf("gh run list: %w", err)
	}
	var runs []struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out, &runs); err != nil {
		return "", fmt.Errorf("gh run list: %w", err)
	}
	for _, r := range runs {
		switch r.Status {
		case "completed":
		case "in_progress":
			return fmt.Sprintf("a pr-agent run is still running on %s", short(f.HeadSHA)), nil
		default: // queued, waiting, requested, pending
			return fmt.Sprintf("a pr-agent run is still queued on %s", short(f.HeadSHA)), nil
		}
	}

	status, ok, err := attestationStatusTime(ctx, o, f.HeadSHA)
	if err != nil || !ok {
		return "", err
	}
	newest, err := newestWord(gh, number)
	if err != nil || !newest.After(status) {
		return "", err
	}
	return fmt.Sprintf("the newest comment or review on the PR (%s) is newer than the %s status (%s), which the workflow has not rewritten since",
		newest.UTC().Format("15:04:05Z"), attestationCheck, status.UTC().Format("15:04:05Z")), nil
}

// attestationStatusTime is when the newest review-attestation status on the
// head was written. ok is false when the head has none.
func attestationStatusTime(ctx context.Context, o Options, sha string) (t time.Time, ok bool, err error) {
	repo := o.Repo
	if repo == "" {
		repo = "{owner}/{repo}" // gh api fills these from the current repository
	}
	out, err := o.Run(ctx, "api", "repos/"+repo+"/commits/"+sha+"/statuses?per_page=100")
	if err != nil {
		return t, false, fmt.Errorf("read the statuses of %s: %w", short(sha), err)
	}
	var statuses []struct {
		Context   string    `json:"context"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.Unmarshal(out, &statuses); err != nil {
		return t, false, fmt.Errorf("read the statuses of %s: %w", short(sha), err)
	}
	for _, s := range statuses {
		if s.Context == attestationCheck && s.CreatedAt.After(t) {
			t, ok = s.CreatedAt, true
		}
	}
	return t, ok, nil
}

// newestWord is the time of the last comment or review on the PR.
func newestWord(gh ghFunc, number int) (time.Time, error) {
	var newest time.Time
	out, err := gh("pr", "view", strconv.Itoa(number), "--json", "comments,reviews")
	if err != nil {
		return newest, fmt.Errorf("gh pr view comments: %w", err)
	}
	var v struct {
		Comments []struct {
			CreatedAt time.Time `json:"createdAt"`
		} `json:"comments"`
		Reviews []struct {
			SubmittedAt time.Time `json:"submittedAt"`
		} `json:"reviews"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return newest, fmt.Errorf("gh pr view comments: %w", err)
	}
	for _, c := range v.Comments {
		if c.CreatedAt.After(newest) {
			newest = c.CreatedAt
		}
	}
	for _, r := range v.Reviews {
		if r.SubmittedAt.After(newest) {
			newest = r.SubmittedAt
		}
	}
	return newest, nil
}
