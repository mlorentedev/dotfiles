package prreview

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// A push to a pull request is reviewed incrementally (/review -i), and only once
// Threshold non-merge commits are newer than the previous review (TOOL-023,
// ADR-040). Below the bound it is not reviewed. Whenever the baseline cannot be
// trusted it asks for a FULL review: PR-Agent's own incremental range is not safe
// to hand a review it did not earn.
//
// "The previous review" is the one PR-Agent itself picks (see the audited source
// identities in harness/pr-agent-upstream-contract.json,
// github_provider.get_previous_review and comment_identity.comment_matches_identity):
// the LAST issue comment carrying a review identity line within its first 5
// lines, or starting with a Guide heading, by BODY alone, with no author check.
// This gate trusts only a review posted by github-actions[bot] (CWE-345: on a
// public repo anyone can comment the same text), and if a comment from anyone
// else, after that review, carries one, the push is reviewed in full: PR-Agent
// would pick that comment as ITS OWN baseline too.
//
// A commit counts as new past that review's HEAD SHA, by POSITION in the commit
// list, which survives a rebase that preserves author dates. If the sha is no
// longer in the PR, the range cannot be trusted: review in full. If no state
// block carries one, as on every incremental review, it counts by date, the
// committer date first as PR-Agent's _commit_timeline_date does. A rebase
// rewrites committer dates, so that errs toward reviewing.
//
// Counting by position can disagree with PR-Agent's get_commit_range, which walks
// by date: if every commit after the reviewed sha is dated at or before the
// review, PR-Agent computes an empty range and publishes nothing. `dotf pr guard`
// fails that run loudly, naming /review as the remedy.
//
// It fails OPEN: input it cannot read means a full review. Skipping one has to be
// justified by the data; running one costs only inference.

const (
	gateBot         = "github-actions[bot]"
	gateStateMarker = "pr-agent-review-state:v1"

	gateHeadingFull        = "## PR Reviewer Guide"
	gateHeadingIncremental = "## Incremental PR Reviewer Guide"

	gateCommentsUnreadable = "the PR comments cannot be read, so the push is reviewed"
	gateCommitsUnreadable  = "the PR commits cannot be read, so the push is reviewed"
)

// Each review identity has two stored forms (hidden_marker_forms): the HTML
// comment, and a link reference for providers that escape HTML comments.
var (
	gateIdentitiesFull = []string{"<!-- pr-agent:review:full -->",
		"[pr-agent:review:full]: https://github.com/The-PR-Agent/pr-agent"}
	gateIdentitiesIncremental = []string{"<!-- pr-agent:review:incremental -->",
		"[pr-agent:review:incremental]: https://github.com/The-PR-Agent/pr-agent"}
)

// GateInputs are the PR's issue comments and commits as the REST API returns
// them, one JSON array each; nil is a file that could not be read. Threshold is
// the number of new non-merge commits that earns a review, and SenderType the
// push event's sender.type.
type GateInputs struct {
	Comments, Commits []byte
	Threshold         int
	SenderType        string
}

// GateDecision is the gate's answer. Mode is empty when Run is false.
type GateDecision struct {
	Run          bool
	Reason, Mode string
}

// Output prints the decision for $GITHUB_OUTPUT. The reason can quote a sha read
// from a comment, and a line break in it would be a second output: the last
// run= wins.
func (d GateDecision) Output() string {
	reason := strings.NewReplacer("\r", " ", "\n", " ").Replace(d.Reason)
	out := fmt.Sprintf("run=%t\nreason=%s\n", d.Run, reason)
	if d.Mode != "" {
		out += "mode=" + d.Mode + "\n"
	}
	return out
}

type gateComment struct {
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	CreatedAt string `json:"created_at"`
	Body      string `json:"body"`
}

type gateCommit struct {
	SHA     string            `json:"sha"`
	Parents []json.RawMessage `json:"parents"`
	Commit  struct {
		Author    struct{ Date string } `json:"author"`
		Committer struct{ Date string } `json:"committer"`
	} `json:"commit"`
}

func gateFull(reason string) GateDecision {
	return GateDecision{Run: true, Reason: reason, Mode: "full"}
}

// Gate decides whether a push is reviewed.
func Gate(in GateInputs) GateDecision {
	if in.SenderType == "Bot" {
		return GateDecision{Reason: "a bot pushed, and PR-Agent skips bot pushes (push_trigger_ignore_bot_commits)"}
	}
	comments, ok := gateList[gateComment](in.Comments)
	if !ok {
		return gateFull(gateCommentsUnreadable)
	}
	baseline, forged := gateBaseline(comments)
	if baseline == nil {
		return gateFull("no previous review, so PR-Agent reviews the whole PR")
	}
	// A date the API did not send is not an old date: unset, every commit is
	// newer and a skip would rest on nothing.
	if baseline.CreatedAt == "" {
		return gateFull(gateCommentsUnreadable)
	}
	if forged {
		return gateFull(fmt.Sprintf("a review marker from a non-bot commenter is newer than the review of %s by %s; "+
			"PR-Agent's own incremental baseline is not author-checked either, so this is reviewed in full",
			baseline.CreatedAt, gateBot))
	}
	commits, ok := gateList[gateCommit](in.Commits)
	if !ok {
		return gateFull(gateCommitsUnreadable)
	}
	if sha := gateReviewedSHA(baseline.Body); sha != "" {
		at := slices.IndexFunc(commits, func(c gateCommit) bool { return c.SHA == sha })
		if at < 0 {
			return gateFull(fmt.Sprintf("the reviewed commit (%s) is no longer in this PR, most likely a rebase or force-push", sha))
		}
		return gateBound(gateNonMerge(commits[at+1:]), in.Threshold, "the reviewed commit "+sha)
	}
	// Every incremental review, and any full one whose block has no head_sha.
	var newer []gateCommit
	for _, c := range gateNonMerge(commits) {
		date := c.Commit.Committer.Date
		if date == "" {
			date = c.Commit.Author.Date
		}
		if date == "" {
			return gateFull(gateCommitsUnreadable)
		}
		if date > baseline.CreatedAt {
			newer = append(newer, c)
		}
	}
	return gateBound(newer, in.Threshold, "the review of "+baseline.CreatedAt)
}

// gateList decodes one JSON array; anything else is unreadable. The workflow
// merges the pages with `jq -s 'add // []'`, so an empty PR is still `[]`.
func gateList[T any](raw []byte) ([]T, bool) {
	var list []T
	if json.Unmarshal(raw, &list) != nil || list == nil {
		return nil, false
	}
	return list, true
}

// gateBaseline is the last review by github-actions[bot], and whether anyone
// else's marker follows it in the listing, which is how PR-Agent walks it.
func gateBaseline(comments []gateComment) (baseline *gateComment, forged bool) {
	for i := range comments {
		switch c := &comments[i]; {
		case !gateIsReview(c.Body):
		case c.User.Login == gateBot:
			baseline, forged = c, false
		default:
			forged = true
		}
	}
	return baseline, forged
}

func gateIsReview(body string) bool {
	return gateHasIdentity(body, gateIdentitiesFull) || gateHasIdentity(body, gateIdentitiesIncremental) ||
		strings.HasPrefix(body, gateHeadingFull) || strings.HasPrefix(body, gateHeadingIncremental)
}

// gateHasIdentity reports an identity among body's first 5 lines, after trimming.
func gateHasIdentity(body string, identities []string) bool {
	lines := strings.SplitN(body, "\n", 6)
	return slices.ContainsFunc(lines[:min(len(lines), 5)], func(line string) bool {
		return slices.Contains(identities, strings.TrimSpace(line))
	})
}

// gateReviewedSHA is last_run.head_sha of the state block ending a FULL review,
// or "" when there is none to trust.
//
// The block is the last thing PR-Agent appends to a full review
// (append_review_state), so everything after the marker, less the closing "-->",
// is its JSON. Anything else carrying the marker is text the review quoted, which
// a PR author can supply: a block in an incremental review, which never appends
// one, or one with text after it. Trusting that would let the author point
// head_sha at their newest commit and skip the review. The keys are read from a
// map because encoding/json matches struct fields case-insensitively, and this
// JSON is not the API's.
func gateReviewedSHA(body string) string {
	if !gateHasIdentity(body, gateIdentitiesFull) && !strings.HasPrefix(body, gateHeadingFull) {
		return ""
	}
	at := strings.LastIndex(body, gateStateMarker)
	if at < 0 {
		return ""
	}
	tail, closed := strings.CutSuffix(strings.TrimSpace(body[at+len(gateStateMarker):]), "-->")
	if !closed {
		return ""
	}
	var state map[string]any
	if json.Unmarshal([]byte(tail), &state) != nil {
		return ""
	}
	lastRun, _ := state["last_run"].(map[string]any)
	sha, _ := lastRun["head_sha"].(string)
	return sha
}

func gateNonMerge(commits []gateCommit) []gateCommit {
	var kept []gateCommit
	for _, c := range commits {
		if len(c.Parents) < 2 {
			kept = append(kept, c)
		}
	}
	return kept
}

// gateBound holds the count of new commits to the threshold.
func gateBound(newer []gateCommit, threshold int, since string) GateDecision {
	if len(newer) >= threshold {
		return GateDecision{Run: true, Mode: "incremental", Reason: fmt.Sprintf(
			"%d new commit(s) since %s, at or past the threshold of %d", len(newer), since, threshold)}
	}
	return GateDecision{Reason: fmt.Sprintf("%d new commit(s) since %s, below the threshold of %d; "+
		"comment /review to ask for one now", len(newer), since, threshold)}
}
