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
	Exempt struct {
		Signatures []Signature `json:"signatures"`
	} `json:"exempt"`
	Classification Classification `json:"classification"`
}

// Classification declares the line a reviewer writes to state a finding's
// class (TOOL-017 AC3): Prefix followed by one of Classes, alone on its line.
// .pr_agent.toml asks the reviewer for exactly these lines; the registry, not
// this package, is where they are declared.
type Classification struct {
	Prefix  string   `json:"prefix"`
	Classes []string `json:"classes"`
	// Reviewers are the logins asked for the marker. A finding from anyone
	// else carries no class by design and is not reported for lacking one.
	Reviewers []string `json:"reviewers"`
}

// Unclassified is the class of a finding that carries no declared marker, or
// two that disagree. It goes to a human; it never decays into a skip.
const Unclassified = "unclassified"

// Signature is a diff that carries nothing a review could act on, declared in
// the registry and honoured by both consumers: the attestation gate reports it
// exempt, and the queue does not list it (#1196). A release PR is the case:
// release-please regenerates it on every merge to main, CodeRabbit re-edits its
// "Reviews paused" notice each time, and the queue read every edit as new
// review output.
type Signature struct {
	Name  string   `json:"name"`
	Files []string `json:"files"`
}

// Reviewer is one declared reviewer. ReviewMarkers are the headings it uses for
// review output published through the comments API. A formal review needs no
// marker: PR.Reviews counts by login, and one reviewer may use both channels.
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
	// Path and Line locate an inline review comment; both are zero for a
	// conversation comment or a review.
	Path string `json:"path,omitempty"`
	Line int    `json:"line,omitempty"`
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
	// Reviews are the formal reviews submitted through the reviews API, each
	// as a Comment stamped with its submission time. A review is review output
	// by its type, so no marker is needed to count it (#1686).
	Reviews []Comment `json:"reviews,omitempty"`
	// Files is the diff's file list, or nil when it was not read. Only a PR
	// the queue would otherwise report has it read (see fetchWith).
	Files []string `json:"files,omitempty"`
}

// Status is the verdict for one pull request.
type Status struct {
	PR       PR
	Pending  bool
	Reviewer string    // who produced the untriaged output
	At       time.Time // when they produced it
	Reason   string    // human-readable, always set

	// Unclassified are the inline findings, newer than the last triage, that a
	// classifying reviewer left without a declared class (TOOL-017 AC3). They
	// are reported for a human and never change Pending.
	Unclassified []Comment
	// FindingsUnread is set when the inline findings could not be read, so an
	// empty Unclassified does not read as "every finding is classified".
	FindingsUnread bool
}

// RegistryPath is where a repository keeps its reviewer registry.
const RegistryPath = "harness/review-attestation.json"

// LoadRegistry reads the reviewer registry from disk.
func LoadRegistry(path string) (Registry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Registry{}, fmt.Errorf("read reviewer registry: %w", err)
	}
	return ParseRegistry(raw, path)
}

// ParseRegistry checks a registry's bytes, wherever they came from; source
// names them in the errors.
func ParseRegistry(raw []byte, source string) (Registry, error) {
	var r Registry
	if err := json.Unmarshal(raw, &r); err != nil {
		return r, fmt.Errorf("parse reviewer registry %s: %w", source, err)
	}
	if strings.TrimSpace(r.Triage.Marker) == "" {
		// Fail toward "cannot tell", never toward "nothing pending". An empty
		// marker would match every comment and silently empty the queue, which
		// is the shape of failure this repo has spent a week cataloguing.
		return r, fmt.Errorf("reviewer registry %s declares no triage.marker", source)
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

// reviewOutput returns the newest output on pr from a declared reviewer: a
// comment carrying one of that reviewer's review markers, or any formal review
// it submitted.
func reviewOutput(pr PR, reg Registry) (Comment, string, bool) {
	var newest Comment
	var by string
	found := false
	consider := func(c Comment, login string) {
		if !found || c.Spoken().After(newest.Spoken()) {
			newest, by, found = c, login, true
		}
	}
	for _, rv := range reg.Reviewers {
		for _, c := range pr.Comments {
			if normaliseLogin(c.Author) == normaliseLogin(rv.Login) && hasMarker(c.Body, rv.ReviewMarkers) {
				consider(c, rv.Login)
			}
		}
		for _, r := range pr.Reviews {
			if normaliseLogin(r.Author) == normaliseLogin(rv.Login) {
				consider(r, rv.Login)
			}
		}
	}
	return newest, by, found
}

func hasMarker(body string, markers []string) bool {
	for _, m := range markers {
		if m != "" && strings.Contains(body, m) {
			return true
		}
	}
	return false
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
	if name, ok := exemptAs(pr.Files, reg); ok {
		return Status{PR: pr, Reason: fmt.Sprintf("exempt: the %q diff signature, nothing here is reviewable", name)}
	}
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

// Classify returns the class body declares through reg's classification
// marker: a line equal, after trimming, to Prefix + one of Classes. The class
// is never read out of prose, so a body that only says "(REAL)" or opens with
// "REAL." is Unclassified however plain it looks.
func Classify(body string, reg Registry) string {
	decl := reg.Classification
	if strings.TrimSpace(decl.Prefix) == "" {
		return Unclassified
	}
	found := ""
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		for _, class := range decl.Classes {
			if line != decl.Prefix+class {
				continue
			}
			// Two classes that disagree are a conflict, not a choice: picking
			// the first would let the order of the reviewer's blocks decide.
			if found != "" && found != class {
				return Unclassified
			}
			found = class
		}
	}
	if found == "" {
		return Unclassified
	}
	return found
}

// UnclassifiedFindings returns the inline findings on pr that need a human
// because their class is not declared: written by a reviewer the registry asks
// for the marker, newer than the last triage, and unclassified by Classify.
func UnclassifiedFindings(pr PR, inline []Comment, reg Registry) []Comment {
	asked := map[string]bool{}
	for _, login := range reg.Classification.Reviewers {
		asked[normaliseLogin(login)] = true
	}
	triaged, any := lastTriage(pr, reg.Triage.Marker)
	var out []Comment
	for _, c := range inline {
		if !asked[normaliseLogin(c.Author)] || (any && !c.Spoken().After(triaged)) {
			continue
		}
		if Classify(c.Body, reg) == Unclassified {
			out = append(out, c)
		}
	}
	return out
}

// exemptAs returns the signature whose files are exactly the diff's: set
// equality in both directions, the attestation gate's rule, so touching one
// file more or one fewer ends the exemption. A diff whose files were not read
// is never exempt.
func exemptAs(files []string, reg Registry) (string, bool) {
	if len(files) == 0 {
		return "", false
	}
	changed := setOf(files)
	for _, sig := range reg.Exempt.Signatures {
		want := setOf(sig.Files)
		if len(want) == 0 || len(want) != len(changed) {
			continue
		}
		same := true
		for f := range changed {
			if !want[f] {
				same = false
				break
			}
		}
		if same {
			return sig.Name, true
		}
	}
	return "", false
}

func setOf(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
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
