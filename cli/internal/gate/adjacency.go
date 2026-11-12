package gate

import (
	"regexp"
	"strings"
)

// The advisory adjacency report (HARNESS-063): other open issues naming a file
// a pull request changes. It never gates. #851 fixed #850 while #849 sat open
// on the same file's other input shape, and nothing connected the two (#857).

// genericBasenames are too common for a name match to carry information.
var genericBasenames = map[string]bool{
	"README.md": true, "AGENTS.md": true, "CLAUDE.md": true, "CHANGELOG.md": true,
	"Makefile": true, "Dockerfile": true, "go.mod": true, "go.sum": true,
}

// Adjacent is one open issue the report names: its number, the path or
// basename it matched on, and its flattened title and body.
type Adjacent struct{ Issue, Match, Text string }

// AdjacencyCandidates is the changed files worth matching on: the production
// ones. Matching the whole diff drags in every issue that names proposal.md
// or docs/lessons.md, and a wall of noise is read once and then ignored.
func AdjacencyCandidates(paths []string) []string {
	var out []string
	for _, p := range paths {
		if p != "" && !ExcludedFromLOC(p) && !IsActiveSpecPath(p) {
			out = append(out, p)
		}
	}
	return out
}

// ContainsWord reports whether word occurs in text bounded by anything but a
// letter, a digit or an underscore: a changed ssh/config must not flag an
// issue that says "configuration".
func ContainsWord(text, word string) bool {
	re := regexp.MustCompile(`(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(word) + `([^A-Za-z0-9_]|$)`)
	return re.MatchString(text)
}

// AdjacentIssues reads the feed, one "<number><TAB><title and body>" per line,
// and returns the issues that name a candidate file, first match per issue,
// skipping the ones this pull request closes. It matches the text as written,
// code spans included: #849 cited its path only inside one, and stripping
// them, as the closing-reference reader must, would hide it.
func AdjacentIssues(feed string, files, closed []string) []Adjacent {
	skip := map[string]bool{}
	for _, n := range closed {
		skip[n] = true
	}
	var out []Adjacent
	for _, line := range strings.Split(feed, "\n") {
		num, text, _ := strings.Cut(strings.Trim(line, "\t"), "\t")
		text = StripControl(strings.Trim(text, "\t"))
		if num == "" || text == "" || skip[num] {
			continue
		}
		if m := firstMatch(text, files); m != "" {
			out = append(out, Adjacent{Issue: num, Match: m, Text: text})
		}
	}
	return out
}

// firstMatch tries the whole path as a plain substring, as the script does: a
// path with a directory in it rarely occurs by accident, so only the bare
// basename needs the word boundary.
func firstMatch(text string, files []string) string {
	for _, f := range files {
		if strings.Contains(text, f) {
			return f
		}
		base := f[strings.LastIndex(f, "/")+1:]
		if !genericBasenames[base] && ContainsWord(text, base) {
			return base
		}
	}
	return ""
}

// Clip is s cut to at most n bytes, backed off to the last whole character.
// On ASCII text it is the script's printf '%.88s'. Where the cut lands inside
// a character the shells already disagree with each other, bash counting
// bytes and zsh characters, and a cut mid-rune would put invalid UTF-8 in an
// annotation and a Markdown table: a whole character is the better answer.
func Clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && n < len(s) && s[n]&0xC0 == 0x80 {
		n--
	}
	return s[:n]
}

// StripControl replaces every ASCII control character with a space. Issue
// titles are written by anyone who can open an issue, and the report prints them
// to stdout, where the runner reads workflow commands, and to the step summary:
// a carriage return would end one command and start another (#3389). The feed
// is line-based, so a newline never arrives; a tab does, and becomes a space.
func StripControl(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
}
