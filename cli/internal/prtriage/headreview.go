package prtriage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// reviewJob is the name of the reviewer's job. A repository that runs it in its
// own workflow reports the check as `review` (ts-bridge#722); one that calls
// the reusable workflow reports `<caller job> / review` (kubelab, hive).
const reviewJob = "review"

// IsReviewCheck reports whether a check run is the reviewer's, under either of
// the names GitHub gives the job. It is the one place both `triage-queue` and
// `pr land` ask, so the two commands cannot disagree about which check covers
// the head (#3243).
func IsReviewCheck(name string) bool {
	return name == reviewJob || strings.HasSuffix(name, " / "+reviewJob)
}

// ReviewAbsent is the reading of a head that carries no review check at all.
const ReviewAbsent = "absent"

// restCheckRun is one item of `GET /repos/{owner}/{repo}/commits/{sha}/check-runs`.
type restCheckRun struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Conclusion string    `json:"conclusion"`
	StartedAt  time.Time `json:"started_at"`
}

// fetchHeadReview reads what the review check says on one head: its
// conclusion once it completed, its status while it has not, or ReviewAbsent.
// The newest run wins, because a cancelled run is superseded by a later one on
// the same head (#3402). A full page is refused, like every other read here: a
// review run past it could be the newest.
func fetchHeadReview(ctx context.Context, run ghRunner, base, sha string) (string, error) {
	if sha == "" {
		return "", fmt.Errorf("the head commit is unknown, so whether a review ran on it cannot be read")
	}
	out, err := run(ctx, "api", fmt.Sprintf("%s/commits/%s/check-runs?per_page=%d", base, sha, checkRunLimit))
	if err != nil {
		return "", fmt.Errorf("gh api check runs for %s: %w", short(sha), err)
	}
	var wire struct {
		CheckRuns []restCheckRun `json:"check_runs"`
	}
	if err := json.Unmarshal(out, &wire); err != nil {
		return "", fmt.Errorf("parse check runs for %s: %w", short(sha), err)
	}
	if len(wire.CheckRuns) >= checkRunLimit {
		return "", fmt.Errorf("GitHub returned %d check runs for %s (a full page); the review check may be past it", checkRunLimit, short(sha))
	}
	var newest *restCheckRun
	for i, c := range wire.CheckRuns {
		if IsReviewCheck(c.Name) && (newest == nil || c.StartedAt.After(newest.StartedAt)) {
			newest = &wire.CheckRuns[i]
		}
	}
	switch {
	case newest == nil:
		return ReviewAbsent, nil
	case newest.Status != "completed":
		return newest.Status, nil
	}
	return newest.Conclusion, nil
}

// withHeadReview holds a PR the comments left clear when no review covers its
// head. Nothing to triage is not the same as a review that found nothing: a
// failed, skipped or missing review leaves the head unreviewed, and the queue
// must not read clear for it (#3243, ts-bridge#687). A draft is never held for
// it: it is not asking to land, and `pr land` refuses it anyway.
func withHeadReview(st Status) Status {
	pr := st.PR
	if st.Pending || pr.Draft || pr.HeadReview == "" || pr.HeadReview == "success" {
		return st
	}
	st.Pending, st.HeadUnreviewed = true, true
	st.Reason = fmt.Sprintf("no review covers the head: the review check on %s reads %s", short(pr.HeadSHA), pr.HeadReview)
	return st
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
