package gate

import (
	"regexp"
	"strings"
)

// StripMarkup returns the body as GitHub renders it for reading, without
// fenced code blocks and HTML comments, so neither a section quoted as an
// example nor a template's guidance counts as the author's answer. Inline code
// stays: paths are usually written as `docs/lessons/...`.
//
// One left-to-right pass, as GitHub reads it, because each construct can hold
// the other's opener: a fence marker inside a comment opens nothing, and a
// `<!--` inside a fence hides nothing (#2310). openFence is the 1-based line of
// a fence that never closes, or 0: GitHub renders everything after one as
// code, and the author needs to be told why a section below it is absent.
//
// A port of scripts/check-knowledge-gate.sh's _strip_markup, held to it by
// tests/check-knowledge-gate.bats while both exist.
func StripMarkup(body string) (text string, openFence int) {
	s := stripper{}
	var out []string
	for n, line := range strings.Split(body, "\n") {
		if kept, ok := s.line(line, n+1); ok {
			out = append(out, kept)
		}
	}
	if s.fence != "" {
		openFence = s.opened
	}
	return strings.Join(out, "\n"), openFence
}

// A fence marker: up to three spaces, then three or more ` or ~. Spaces, not
// tabs: CommonMark reads a tab as four columns, an indented code line.
var fenceRe = regexp.MustCompile("^ ? ? ?(`{3,}|~{3,})")

var blankRe = regexp.MustCompile(`^[ \t]*$`)

type stripper struct {
	inside bool   // inside an HTML comment
	fence  string // the open fence's marker, or ""
	opened int    // the line the open fence opened on
}

// line returns what a line contributes to the text, and false for a line that
// is all fence.
func (s *stripper) line(line string, n int) (string, bool) {
	if loc := fenceRe.FindStringIndex(line); !s.inside && loc != nil {
		marker := strings.TrimLeft(line[:loc[1]], " ")
		rest := line[loc[1]:]
		if s.fence != "" {
			// Only a bare marker closes: the opener's character, at least as
			// long, and nothing after it but blanks.
			if marker[0] == s.fence[0] && len(marker) >= len(s.fence) && blankRe.MatchString(rest) {
				s.fence = ""
			}
			return "", false
		}
		// A backtick fence cannot carry a backtick in its info string: such a
		// line is inline code in a paragraph, not an opener.
		if marker[0] != '`' || !strings.Contains(rest, "`") {
			s.fence, s.opened = marker, n
			return "", false
		}
	}
	if s.fence != "" {
		return "", false
	}
	return s.stripComments(line), true
}

// stripComments drops HTML comments from a line, carrying an open comment
// across lines. A code span, a backtick run closed by a run of the same
// length, is kept as text: a <!-- inside it opens nothing.
func (s *stripper) stripComments(line string) string {
	var out strings.Builder
	for line != "" {
		if s.inside {
			i := strings.Index(line, "-->")
			if i < 0 {
				return out.String()
			}
			line, s.inside = line[i+3:], false
			continue
		}
		i, j := strings.Index(line, "<!--"), strings.IndexByte(line, '`')
		if j >= 0 && (i < 0 || j < i) {
			kept, rest := codeSpan(line, j)
			out.WriteString(kept)
			line = rest
			continue
		}
		if i < 0 {
			out.WriteString(line)
			return out.String()
		}
		out.WriteString(line[:i])
		line, s.inside = line[i+4:], true
	}
	return out.String()
}

// codeSpan splits a line at the backtick at j: what is kept as text through
// the end of the span, and what is left to scan. A backtick escaped by an odd
// number of backslashes is literal, and so is a run with no closer on the line.
func codeSpan(line string, j int) (kept, rest string) {
	if escaped(line, j) {
		return line[:j+1], line[j+1:]
	}
	run := runAt(line, j)
	after := line[j+run:]
	k := spanEnd(after, run)
	return line[:j+run] + after[:k], after[k:]
}

// runAt is the length of the backtick run starting at i.
func runAt(s string, i int) int {
	r := 0
	for i+r < len(s) && s[i+r] == '`' {
		r++
	}
	return r
}

// escaped reports whether the byte at i follows an odd number of backslashes.
// An even number is literal backslashes, escaping nothing.
func escaped(s string, i int) bool {
	b := 0
	for i-b > 0 && s[i-b-1] == '\\' {
		b++
	}
	return b%2 == 1
}

// spanEnd is the length of s through the first run of exactly n backticks,
// or 0 when there is none.
func spanEnd(s string, n int) int {
	for i := 0; i < len(s); {
		if s[i] != '`' {
			i++
			continue
		}
		r := runAt(s, i)
		if r == n {
			return i + n
		}
		i += r
	}
	return 0
}
