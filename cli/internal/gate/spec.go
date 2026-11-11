package gate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// The spec gate's pure parts, ported from scripts/check-spec-gate.sh: which
// issues a pull request closes, which spec an issue belongs to, and which
// changed lines count as production code. Each mirrors its shell function,
// named in its comment, until dotf gate spec replaces the script.

var (
	// Up to three blanks, tabs included, as the shell's [[:space:]]{0,3}.
	specFenceRe = regexp.MustCompile("^[[:space:]]{0,3}(`{3,}|~{3,})")
	codeSpanRe  = regexp.MustCompile("`[^`]*`")
)

// StripCode is the body without fenced blocks and inline code spans
// (_strip_markdown_code). GitHub does not close an issue from a reference
// inside code, so neither may the gate.
//
// It differs from StripMarkup on purpose, for now: a fence closes on any
// marker of the opener's character at least as long, info string or not, and
// inline spans are removed rather than kept (#2407).
func StripCode(body string) string {
	var out []string
	fence := ""
	for _, line := range strings.Split(body, "\n") {
		if m := specFenceRe.FindStringSubmatch(line); m != nil {
			marker := m[1]
			if fence == "" {
				fence = marker
			} else if marker[0] == fence[0] && len(marker) >= len(fence) {
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		out = append(out, codeSpanRe.ReplaceAllString(line, ""))
	}
	// The shell's command substitution drops trailing newlines.
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

var (
	issueURLRe = regexp.MustCompile(`https?://github\.com/([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)/issues/([0-9]+)`)
	closingRe  = regexp.MustCompile(`(?i)\b(close[sd]?|fix(e[sd])?|resolve[sd]?):?[[:space:]]+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)?#[0-9]+`)
)

// ClosingIssueNumbers is every issue number the text closes in this
// repository, in order of appearance (_closing_issue_numbers). Only GitHub's
// closing verbs count. A qualified reference counts only when it names slug;
// with no slug, ownership cannot be proven and every qualified one is ignored.
func ClosingIssueNumbers(slug, text string) []string {
	text = issueURLRe.ReplaceAllString(StripCode(text), "$1#$2")
	var nums []string
	// Line by line, as grep reads: a keyword never pairs with a number on the
	// next line.
	for _, line := range strings.Split(text, "\n") {
		for _, match := range closingRe.FindAllString(line, -1) {
			fields := strings.Fields(match)
			qualifier, number, _ := strings.Cut(fields[len(fields)-1], "#")
			if qualifier != "" && (slug == "" || qualifier != slug) {
				continue
			}
			nums = append(nums, number)
		}
	}
	return nums
}

// FrontmatterIssue is the issue number a proposal.md's frontmatter declares,
// or "" (_frontmatter_issue_number). Three shapes exist: "owner/repo#N",
// "repo#N" and a bare N, the quoted ones followed by a comment.
func FrontmatterIssue(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || !isFrontmatterFence(lines[0]) {
		return ""
	}
	for _, l := range lines[1:] {
		if isFrontmatterFence(l) {
			return ""
		}
		v, ok := strings.CutPrefix(l, "issue:")
		if !ok {
			continue
		}
		v = strings.TrimLeft(v, " \t")
		if rest, quoted := strings.CutPrefix(v, `"`); quoted {
			v, _, _ = strings.Cut(rest, `"`)
		} else if i := strings.IndexAny(v, " \t"); i >= 0 {
			v = v[:i]
		}
		if i := strings.LastIndex(v, "#"); i >= 0 {
			v = v[i+1:]
		}
		if isDigits(v) {
			return v
		}
		return ""
	}
	return ""
}

func isFrontmatterFence(l string) bool {
	return strings.TrimRight(l, " \t\r") == "---"
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ExcludedFromLOC reports whether a changed path is not production code
// (_excluded): prose under docs/, tests, archived specs, a run's report,
// Go tests, lockfiles and patch-tool output. Never a substring match such as
// "generated": that once excluded hand-written code.
func ExcludedFromLOC(path string) bool {
	switch {
	case strings.HasPrefix(path, "docs/") && strings.HasSuffix(path, ".md"),
		strings.HasPrefix(path, "tests/"),
		strings.HasPrefix(path, "specs/archive/"):
		return true
	}
	if run, ok := strings.CutPrefix(path, ".agent/runs/"); ok && strings.HasSuffix(run, "/report.md") {
		// The report of one run, not a deeper path ending the same way.
		return strings.Count(run, "/") < 2
	}
	base := path[strings.LastIndex(path, "/")+1:]
	switch base {
	case "package-lock.json", "pnpm-lock.yaml", "go.sum", ".gitignore", "CHANGELOG.md":
		return true
	}
	for _, suffix := range []string{"_test.go", ".lock", ".lockb", ".patch", ".diff", ".orig", ".rej"} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	return false
}

// IsActiveSpecPath reports a path inside an active spec folder,
// specs/<id>/..., not specs/archive/ (_is_active_spec_path).
func IsActiveSpecPath(path string) bool {
	rest, ok := strings.CutPrefix(path, "specs/")
	return ok && !strings.HasPrefix(path, "specs/archive/") && strings.Contains(rest, "/")
}

var braceRenameRe = regexp.MustCompile(`\{[^}]* => ([^}]*)\}`)

// NormalizeRenamePath turns git's compressed rename, "a/{old => new}/b" or
// "old => new", into the destination path (_normalize_rename_path), so an
// archive move is judged where it lands.
func NormalizeRenamePath(p string) string {
	if strings.Contains(p, "{") && strings.Contains(p, " => ") {
		p = braceRenameRe.ReplaceAllString(p, "$1")
	}
	if i := strings.LastIndex(p, " => "); i >= 0 {
		p = p[i+len(" => "):]
	}
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	return p
}

// FileStat is one line of git diff --numstat. A binary file counts 0 and 0.
type FileStat struct {
	Path           string
	Added, Removed int
}

// ParseNumstat reads git diff --numstat output, renames resolved to their
// destination. A count that is neither a number nor "-" is an error: the gate
// fails closed rather than count it as nothing.
func ParseNumstat(out string) ([]FileStat, error) {
	var stats []FileStat
	for _, line := range strings.Split(out, "\n") {
		added, rest, _ := strings.Cut(line, "\t")
		removed, path, _ := strings.Cut(rest, "\t")
		if path == "" {
			continue
		}
		a, errA := numstatCount(added)
		r, errR := numstatCount(removed)
		if errA != nil || errR != nil {
			return nil, fmt.Errorf("numstat line %q: counts are not numbers", line)
		}
		stats = append(stats, FileStat{Path: NormalizeRenamePath(path), Added: a, Removed: r})
	}
	return stats, nil
}

func numstatCount(s string) (int, error) {
	if s == "-" {
		return 0, nil
	}
	return strconv.Atoi(s)
}
