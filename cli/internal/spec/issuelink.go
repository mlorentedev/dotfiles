package spec

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// IssueRef names one GitHub issue. Repo is "owner/name".
type IssueRef struct {
	Repo   string
	Number int
}

func (r IssueRef) String() string { return fmt.Sprintf("%s#%d", r.Repo, r.Number) }

// LinkSource records where ResolveIssueLink found a spec's tracking issue.
type LinkSource int

const (
	// LinkNone: the spec names no tracking issue anywhere.
	LinkNone LinkSource = iota
	// LinkFrontmatter: proposal.md's `issue:` field — the contract `spec init`
	// writes and the archive-on-merge gate reads.
	LinkFrontmatter
	// LinkProse: a labelled prose line. Accepted so a pre-gate spec can still
	// be audited, and reported so it gets normalised — never a second SSOT.
	LinkProse
)

func (s LinkSource) String() string {
	switch s {
	case LinkFrontmatter:
		return "frontmatter"
	case LinkProse:
		return "prose"
	}
	return "none"
}

// issueRefPattern matches one reference. The alternatives are tried leftmost
// first at each position, so a URL wins over the `name#N` it does not contain,
// and `owner/name#N` wins over the `name#N` suffix inside it.
var issueRefPattern = regexp.MustCompile(
	`https?://github\.com/([\w.-]+)/([\w.-]+)/issues/(\d+)` +
		`|([\w.-]+)/([\w.-]+)#(\d+)` +
		`|([\w.-]+)#(\d+)` +
		`|#(\d+)`)

// trackingLabel is the grammar of a prose label that names the spec's own
// tracking issue: `issue(s)`, `GH`, `GitHub`, or one qualifier word before
// `issue(s)` (`GH issue`, `hive issue`, `github_issue`).
var trackingLabel = regexp.MustCompile(`^(?:gh|github|(?:[a-z0-9.-]+ )?issues?)$`)

// nonTrackingWords mark a labelled issue that belongs to someone else's
// tracker or is merely adjacent — measured: "Upstream issue:
// anthropics/claude-code#…" (BUG-004), "Related upstream issue: …" (hive
// HIVE-104), "Sister sunset issue: …" (AI-020).
var nonTrackingWords = []string{"upstream", "related", "sister"}

// ResolveIssueLink returns the tracking issue a proposal.md declares, and
// where it was found. homeRepo ("owner/name") completes `name#N` and `#N`.
//
// A non-empty `issue:` field is authoritative and must hold exactly one
// reference; anything else is an error, because a placeholder such as "TBD"
// is an unresolvable link, not an absent one. Only when the field is absent
// or empty is the prose fallback consulted.
func ResolveIssueLink(proposal, homeRepo string) (IssueRef, LinkSource, error) {
	if raw := frontmatterFields(proposal)["issue"]; raw != "" {
		ref, err := parseOneRef(raw, homeRepo)
		return ref, LinkFrontmatter, err
	}
	if ref, ok := proseIssueLink(proposal, homeRepo); ok {
		return ref, LinkProse, nil
	}
	return IssueRef{}, LinkNone, nil
}

func parseOneRef(raw, homeRepo string) (IssueRef, error) {
	raw = strings.TrimSpace(raw)
	loc := issueRefPattern.FindAllStringSubmatchIndex(raw, -1)
	if len(loc) != 1 || loc[0][0] != 0 || loc[0][1] != len(raw) {
		return IssueRef{}, fmt.Errorf("issue: %q is not one issue reference (want owner/name#N, name#N, #N or an issue URL)", raw)
	}
	return refFromMatch(issueRefPattern.FindStringSubmatch(raw), homeRepo)
}

// proseIssueLink returns the first reference, on a line carrying a tracking
// label, whose owner is homeRepo's owner. An unlabelled `#N` in running text
// is never a link: "(GH #197)" and "PR #121" name other things.
func proseIssueLink(proposal, homeRepo string) (IssueRef, bool) {
	for _, line := range strings.Split(proposal, "\n") {
		label, rest, ok := splitLabel(line)
		if !ok || !isTrackingLabel(label) {
			continue
		}
		for _, m := range issueRefPattern.FindAllStringSubmatch(rest, -1) {
			ref, err := refFromMatch(m, homeRepo)
			if err == nil && sameOwner(ref.Repo, homeRepo) {
				return ref, true
			}
		}
	}
	return IssueRef{}, false
}

// splitLabel strips list, quote and bold markers and splits "Label: rest".
func splitLabel(line string) (label, rest string, ok bool) {
	s := strings.TrimLeft(line, " \t>-*+")
	label, rest, ok = strings.Cut(s, ":")
	return strings.Trim(label, " *_`"), rest, ok
}

func isTrackingLabel(label string) bool {
	l := strings.ToLower(strings.ReplaceAll(label, "_", " "))
	for _, w := range nonTrackingWords {
		if strings.Contains(l, w) {
			return false
		}
	}
	return trackingLabel.MatchString(l)
}

func refFromMatch(m []string, homeRepo string) (IssueRef, error) {
	var repo, num string
	switch {
	case m[3] != "":
		repo, num = m[1]+"/"+m[2], m[3]
	case m[6] != "":
		repo, num = m[4]+"/"+m[5], m[6]
	case m[8] != "":
		repo, num = ownerOf(homeRepo)+"/"+m[7], m[8]
	default:
		repo, num = homeRepo, m[9]
	}
	n, err := strconv.Atoi(num)
	if err != nil || n <= 0 {
		return IssueRef{}, fmt.Errorf("issue number %q is not a positive integer", num)
	}
	return IssueRef{Repo: repo, Number: n}, nil
}

func ownerOf(repo string) string {
	owner, _, _ := strings.Cut(repo, "/")
	return owner
}

func sameOwner(a, b string) bool { return strings.EqualFold(ownerOf(a), ownerOf(b)) }
