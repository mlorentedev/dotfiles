package gate

import (
	"regexp"
	"sort"
	"strings"
)

// SpecFloor is the added plus removed lines inside active spec folders a
// change needs for its spec touch to count: a one-line edit to an unrelated
// spec must not legitimise a large pull request (#686).
const SpecFloor = 10

// SkipRationale reports whether the body's "## <heading>" section has any
// non-blank text before the next level-2 heading (_skip_rationale_nonempty).
func SkipRationale(body, heading string) bool {
	head := regexp.MustCompile(`^## ` + regexp.QuoteMeta(heading) + `[[:space:]]*$`)
	in := false
	for _, l := range strings.Split(body, "\n") {
		switch {
		case head.MatchString(l):
			in = true
		case in && strings.HasPrefix(l, "## "):
			return false
		case in && strings.Trim(l, " \t\n\v\f\r") != "":
			return true
		}
	}
	return false
}

// SpecIssue links a spec folder to the issue its proposal declares.
type SpecIssue struct{ Issue, ID string }

// SpecProposalID is the spec id of a path specs/<root><id>/proposal.md, where
// root is "" for an active spec and "archive/" for an archived one, or "".
// A deeper path, such as an abandoned spec's, is not one.
func SpecProposalID(path, root string) string {
	rest, ok := strings.CutPrefix(path, "specs/"+root)
	if !ok || (root == "" && strings.HasPrefix(path, "specs/archive/")) {
		return ""
	}
	id, ok := strings.CutSuffix(rest, "/proposal.md")
	if !ok || id == "" || strings.Contains(id, "/") {
		return ""
	}
	return id
}

// SortedUnique is the strings in byte order without repeats, as sort -u
// under the C locale.
func SortedUnique(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
