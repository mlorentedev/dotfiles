package mem

import (
	"regexp"
	"strings"
)

// ShellQuote quotes a path for a command the user copies: a single-quoted
// word, with any single quote inside it closed, escaped and reopened.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// plainWord is a word the shell reads as itself, so it needs no quotes.
var plainWord = regexp.MustCompile(`^[A-Za-z0-9_./@+-]+$`)

// ShellWord is s as one shell word: as written when it is plain, quoted when
// not. A thread key comes from MEMORY.md, input this session did not write, so
// a heading cannot put a `;` into a command the operator copies (MEMORY-008
// review rounds 4 and 5). Every command line that names a thread key goes
// through it.
func ShellWord(s string) string {
	if plainWord.MatchString(s) {
		return s
	}
	return ShellQuote(s)
}
