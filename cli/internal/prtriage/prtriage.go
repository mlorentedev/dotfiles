// Package prtriage answers one question: which open pull requests carry reviewer
// output that nobody has dispositioned yet.
//
// It exists because the loop had no wake-up. GUARD-002 makes a green check mean
// *reviewed*; nothing made anyone act on the review. CI cannot help here — a
// workflow_run re-evaluates the gate and GitHub notifies the human, but no push
// channel reaches an agent session. So the mechanism an agent can actually use
// is a query it runs at a checkpoint, which is this.
//
// It lists and never applies. The judgement stays in the `pr-review-triage`
// skill, which disposes of each comment under explicit human confirmation —
// that contract is deliberate and this package must not erode it.
package prtriage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Registry is the subset of harness/review-attestation.json this package reads.
// The same file drives the attestation gate's classifier, on purpose: one
// registry with two consumers, rather than two files that agree until they
// don't.
type Registry struct {
	Reviewers []Reviewer `json:"reviewers"`
	Triage    struct {
		Marker string `json:"marker"`
	} `json:"triage"`
}

// Reviewer is one declared reviewer. ReviewMarkers are the headings a reviewer
// that publishes through the comments API uses for its review output; a
// reviewer that files proper reviews carries none.
type Reviewer struct {
	Login         string   `json:"login"`
	ReviewMarkers []string `json:"review_markers"`
}

// Comment is one comment on a pull request, in the shape `gh` returns.
type Comment struct {
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Spoken is when this comment last said something new.
//
// Both reviewers in use here EDIT THEIR COMMENT IN PLACE when they re-review
// after a push: the body grows new findings and `createdAt` does not move
// (TOOL-019, #1422). Measured on #1543 the night this was written --
// CodeRabbit's comment was created 02:52:56 and updated 03:28:29, and the
// review that arrived in that edit carried three real findings.
//
// max() rather than UpdatedAt alone, and that is not tidiness. A comment that
// was never edited can carry a zero UpdatedAt -- every fixture predating this
// field does -- and reading zero as "spoken at the epoch" would make the whole
// queue answer "nothing pending". The wrong direction: this package exists to
// refuse to report an empty queue it did not compute.
func (c Comment) Spoken() time.Time {
	if c.UpdatedAt.After(c.CreatedAt) {
		return c.UpdatedAt
	}
	return c.CreatedAt
}

// PR is one open pull request and everything said on it.
type PR struct {
	Number   int       `json:"number"`
	Title    string    `json:"title"`
	URL      string    `json:"url"`
	Comments []Comment `json:"comments"`
}

// Status is the verdict for one pull request.
type Status struct {
	PR       PR
	Pending  bool
	Reviewer string    // who produced the untriaged output
	At       time.Time // when they produced it
	Reason   string    // human-readable, always set
}

// LoadRegistry reads the reviewer registry from disk.
func LoadRegistry(path string) (Registry, error) {
	var r Registry
	raw, err := os.ReadFile(path)
	if err != nil {
		return r, fmt.Errorf("read reviewer registry: %w", err)
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return r, fmt.Errorf("parse reviewer registry %s: %w", path, err)
	}
	if strings.TrimSpace(r.Triage.Marker) == "" {
		// Fail toward "cannot tell", never toward "nothing pending". An empty
		// marker would match every comment and silently empty the queue, which
		// is the shape of failure this repo has spent a week cataloguing.
		return r, fmt.Errorf("reviewer registry %s declares no triage.marker", path)
	}
	return r, nil
}

// normaliseLogin folds the two spellings GitHub's APIs use for the same actor:
// GraphQL returns `github-actions`, REST returns `github-actions[bot]`. Matching
// the raw string made the attestation gate depend on which API produced the
// payload (#1033); the same trap applies here.
func normaliseLogin(s string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(s), "[bot]"))
}

// reviewOutput returns the newest comment on pr that a declared reviewer wrote
// and that carries one of that reviewer's review markers.
func reviewOutput(pr PR, reg Registry) (Comment, string, bool) {
	var newest Comment
	var by string
	found := false
	for _, c := range pr.Comments {
		for _, rv := range reg.Reviewers {
			if normaliseLogin(c.Author) != normaliseLogin(rv.Login) {
				continue
			}
			for _, m := range rv.ReviewMarkers {
				if m == "" || !strings.Contains(c.Body, m) {
					continue
				}
				if !found || c.Spoken().After(newest.Spoken()) {
					newest, by, found = c, rv.Login, true
				}
			}
		}
	}
	return newest, by, found
}

// lastTriage returns the newest triage record on pr. The marker is matched at
// the start of a line so that quoting it inside prose — this package's own
// documentation, for instance — does not read as a disposition.
//
// CreatedAt here, deliberately, while reviewOutput uses Spoken(). The asymmetry
// is the safe direction of each error:
//
//   - A reviewer's edit is NEW OUTPUT. Missing it reports a clear queue over
//     unread findings, so the reviewer side must count edits.
//   - A triage's edit is not new READING. Counting it would let a typo fix on an
//     old disposition silently clear a queue the reviewer has since added to,
//     and would hand anyone a one-keystroke way to empty the queue without
//     looking at it.
//
// So edits make the queue louder and never quieter. Both errors are possible;
// only one of them is safe, and it is the one that costs a re-read.
func lastTriage(pr PR, marker string) (time.Time, bool) {
	var newest time.Time
	found := false
	for _, c := range pr.Comments {
		if !hasLinePrefix(c.Body, marker) {
			continue
		}
		if !found || c.CreatedAt.After(newest) {
			newest, found = c.CreatedAt, true
		}
	}
	return newest, found
}

func hasLinePrefix(body, marker string) bool {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), marker) {
			return true
		}
	}
	return false
}

// Evaluate decides, for one pull request, whether reviewer output is awaiting a
// disposition. A PR is pending when a reviewer has spoken and either nobody has
// recorded a triage since, or the reviewer has spoken again after the last one —
// which is the case that matters, because pushing a fix makes the reviewer
// re-review and the earlier disposition no longer covers what it said.
func Evaluate(pr PR, reg Registry) Status {
	out, by, ok := reviewOutput(pr, reg)
	if !ok {
		return Status{PR: pr, Reason: "no reviewer output yet"}
	}
	triaged, any := lastTriage(pr, reg.Triage.Marker)
	if !any {
		return Status{PR: pr, Pending: true, Reviewer: by, At: out.Spoken(),
			Reason: fmt.Sprintf("%s reviewed, never triaged", by)}
	}
	if out.Spoken().After(triaged) {
		return Status{PR: pr, Pending: true, Reviewer: by, At: out.Spoken(),
			Reason: fmt.Sprintf("%s reviewed again after the last triage", by)}
	}
	return Status{PR: pr, Reviewer: by, At: out.Spoken(), Reason: "triaged"}
}

// Queue evaluates every pull request and returns those awaiting a disposition.
func Queue(prs []PR, reg Registry) []Status {
	var pending []Status
	for _, pr := range prs {
		if st := Evaluate(pr, reg); st.Pending {
			pending = append(pending, st)
		}
	}
	return pending
}
