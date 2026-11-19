package harness

import (
	"regexp"
	"strings"
)

// word is one shell word after quote removal.
type word struct {
	text string
	// raw is text with unquoted backslashes kept. POSIX drops them, but a model
	// writing `cd C:\Users\u\repo` means the path, so directory arguments read raw.
	raw string
	// dynamic marks a word that carries a variable or substitution and so cannot
	// be evaluated from the command text.
	dynamic bool
	// quoted marks a word that held any quoting, which a redirection never does.
	quoted bool
	// redir marks a word that holds an unquoted `<` or `>`: a redirection,
	// whatever else in it is quoted (`>"out.txt"`).
	redir bool
	// sub marks a word read inside a subshell or a substitution, where a `cd`
	// does not carry out to the commands after it.
	sub bool
}

// command is one simple command: its words in order.
type command []word

// frame remembers the quote state to return to when a `$(`/backtick closes.
type frame struct {
	quote    byte
	backtick bool
	parens   int // open `(` subshells inside this `$(`
}

type lexer struct {
	shell Shell
	cmds  []command
	cur   command
	text  strings.Builder
	raw   strings.Builder
	has   bool
	dyn   bool
	quote bool
	stack []frame
	redir bool
	paren int  // open `(` subshells outside any `$(`
	q     byte // 0, '\'', '"' or 'A' for $'...' (backslash escapes the next rune)
}

// splitCommands breaks a shell string into simple commands. It reads `&&`, `||`,
// `;`, `|`, `&`, newlines, subshell parentheses, `$(...)` and backticks as
// boundaries, and never evaluates anything. ok is false when quoting or a
// substitution is left open: the commands are then best-effort.
func splitCommands(s string, shell Shell) (cmds []command, ok bool) {
	lx := &lexer{shell: shell}
	text := s
	if shell == ShellPOSIX {
		text = stripHeredocs(s)
	}
	rs := []rune(text)
	for i := 0; i < len(rs); i++ {
		i = lx.step(rs, i)
	}
	lx.endCommand()
	return lx.cmds, lx.q == 0 && len(lx.stack) == 0
}

func (lx *lexer) endWord() {
	if lx.has {
		lx.cur = append(lx.cur, word{
			text: lx.text.String(), raw: lx.raw.String(), dynamic: lx.dyn, quoted: lx.quote, redir: lx.redir,
			sub: len(lx.stack) > 0 || lx.paren > 0,
		})
	}
	lx.text.Reset()
	lx.raw.Reset()
	lx.has, lx.dyn, lx.quote, lx.redir = false, false, false, false
}

func (lx *lexer) endCommand() {
	lx.endWord()
	if len(lx.cur) > 0 {
		lx.cmds = append(lx.cmds, lx.cur)
	}
	lx.cur = nil
}

func (lx *lexer) put(r rune) {
	lx.text.WriteRune(r)
	lx.raw.WriteRune(r)
	lx.has = true
}

// step consumes the rune at i (and possibly its successor) and returns the
// index of the last rune consumed.
func (lx *lexer) step(rs []rune, i int) int {
	c := rs[i]
	if lx.q == 'A' && c == '\\' && i+1 < len(rs) {
		lx.put(rs[i+1])
		return i + 1
	}
	if lx.q == '\'' || lx.q == 'A' {
		if c == '\'' {
			lx.q = 0
		} else {
			lx.put(c)
		}
		return i
	}
	if next, handled := lx.escapeOrSubst(rs, i); handled {
		return next
	}
	if lx.q == '"' {
		if c == '"' {
			lx.q = 0
		} else {
			lx.put(c)
		}
		return i
	}
	return lx.unquoted(rs, i)
}

// escapeOrSubst handles what is special both unquoted and inside double quotes.
func (lx *lexer) escapeOrSubst(rs []rune, i int) (int, bool) {
	c := rs[i]
	switch {
	case c == '$':
		return lx.dollar(rs, i), true
	case c == '`' && lx.shell == ShellPOSIX:
		lx.backtick()
		return i, true
	case c == '\\' && lx.shell == ShellPOSIX:
		return lx.backslash(rs, i), true
	case c == '`' && lx.shell == ShellPowerShell && i+1 < len(rs):
		lx.put(rs[i+1])
		return i + 1, true
	}
	return i, false
}

// dollar handles `$(`, `$'` (ANSI-C quoting, POSIX only) and a plain `$`.
func (lx *lexer) dollar(rs []rune, i int) int {
	next := rune(0)
	if i+1 < len(rs) {
		next = rs[i+1]
	}
	switch {
	case next == '(':
		lx.endCommand()
		lx.stack = append(lx.stack, frame{quote: lx.q})
		lx.q = 0
		return i + 1
	case next == '\'' && lx.q == 0 && lx.shell == ShellPOSIX:
		lx.q, lx.has, lx.quote = 'A', true, true
		return i + 1
	}
	lx.dyn = true
	lx.put('$')
	return i
}

// backtick opens or closes a backtick substitution.
func (lx *lexer) backtick() {
	lx.endCommand()
	if n := len(lx.stack); n > 0 && lx.stack[n-1].backtick {
		lx.q = lx.stack[n-1].quote
		lx.stack = lx.stack[:n-1]
		return
	}
	lx.stack = append(lx.stack, frame{quote: lx.q, backtick: true})
	lx.q = 0
}

func (lx *lexer) backslash(rs []rune, i int) int {
	if i+1 >= len(rs) {
		lx.put('\\')
		return i
	}
	n := rs[i+1]
	if n == '\n' { // a line continuation joins the two lines
		return i + 1
	}
	if n == '\r' && i+2 < len(rs) && rs[i+2] == '\n' {
		return i + 2
	}
	if lx.q == '"' && !strings.ContainsRune("\"\\$`", n) {
		lx.put('\\')
		return i
	}
	lx.text.WriteRune(n)
	lx.raw.WriteRune('\\')
	lx.raw.WriteRune(n)
	lx.has = true
	return i + 1
}

func (lx *lexer) unquoted(rs []rune, i int) int {
	c := rs[i]
	switch {
	case c == '\'' || c == '"':
		lx.q, lx.has, lx.quote = byte(c), true, true
	case c == ' ' || c == '\t' || c == '\r':
		lx.endWord()
	case c == '#' && !lx.has:
		for i+1 < len(rs) && rs[i+1] != '\n' {
			i++
		}
	case c == '<' && lx.shell == ShellPowerShell && i+1 < len(rs) && rs[i+1] == '#':
		lx.endWord()
		i = blockCommentEnd(rs, i)
	default:
		lx.operator(rs, i)
	}
	return i
}

// blockCommentEnd returns the index of the `>` that closes the PowerShell
// `<# ... #>` opened at i, or the last rune when it is never closed.
func blockCommentEnd(rs []rune, i int) int {
	for j := i + 2; j+1 < len(rs); j++ {
		if rs[j] == '#' && rs[j+1] == '>' {
			return j + 1
		}
	}
	return len(rs) - 1
}

// doubled reports whether the `|` or `&` at i is half of `||` or `&&`.
func doubled(rs []rune, i int) bool {
	return (i+1 < len(rs) && rs[i+1] == rs[i]) || (i > 0 && rs[i-1] == rs[i])
}

// markSub marks the words of the command being read as run in a subshell.
func (lx *lexer) markSub() {
	for k := range lx.cur {
		lx.cur[k].sub = true
	}
}

// operator handles what is left once quotes, blanks and comments are out: a
// command boundary, a redirection's `&`, or an ordinary character.
func (lx *lexer) operator(rs []rune, i int) {
	c := rs[i]
	switch {
	case c == '&' && lx.redirectAmp(rs, i):
		lx.put(c)
	case (c == '|' || c == '&') && !doubled(rs, i):
		lx.endWord()
		lx.markSub() // the left side of a pipe or an & runs in a subshell
		lx.boundary(c)
	case strings.ContainsRune(";&|\n()", c), lx.shell == ShellPowerShell && (c == '{' || c == '}'):
		lx.boundary(c)
	default:
		lx.redir = lx.redir || c == '<' || c == '>'
		lx.put(c)
	}
}

// boundary ends the current command. A `)` that closes a `$(` also restores
// the quote state the substitution was opened in.
func (lx *lexer) boundary(c rune) {
	lx.endCommand()
	n := len(lx.stack)
	if n == 0 {
		lx.paren = max(0, lx.paren+map[rune]int{'(': 1, ')': -1}[c])
		return
	}
	if lx.stack[n-1].backtick {
		return
	}
	switch {
	case c == '(':
		lx.stack[n-1].parens++
	case c == ')' && lx.stack[n-1].parens > 0:
		lx.stack[n-1].parens--
	case c == ')':
		lx.q = lx.stack[n-1].quote
		lx.stack = lx.stack[:n-1]
	}
}

// redirectAmp reports whether an `&` belongs to a redirection (`2>&1`, `&>f`)
// rather than starting a background job.
func (lx *lexer) redirectAmp(rs []rune, i int) bool {
	s := lx.text.String()
	if lx.has && (strings.HasSuffix(s, ">") || strings.HasSuffix(s, "<")) {
		return true
	}
	return i+1 < len(rs) && rs[i+1] == '>' && !lx.has
}

// stripRedirects removes redirections (`> f`, `2>&1`, `<in`) so the file name
// is not read as an argument of the command.
func stripRedirects(cmd command) command {
	out := make(command, 0, len(cmd))
	for i := 0; i < len(cmd); i++ {
		w := cmd[i]
		if w.quoted && !w.redir {
			out = append(out, w)
			continue
		}
		if rest, isRedir := redirectRest(w.text); isRedir {
			if rest == "" {
				i++
			}
			continue
		}
		out = append(out, w)
	}
	return out
}

// redirectRest reports whether t starts with a redirection operator and returns
// what follows it in the same word ("" means the target is the next word).
func redirectRest(t string) (string, bool) {
	s := strings.TrimLeft(t, "0123456789")
	for _, op := range []string{"&>>", "&>", ">>", ">&", ">|", "<<<", "<<", "<&", "<>", ">", "<"} {
		if rest, ok := strings.CutPrefix(s, op); ok {
			if (op == ">&" || op == "<&") && rest != "" {
				return "x", true // `2>&1`: the target is part of the word
			}
			return rest, true
		}
	}
	return "", false
}

var (
	heredocStart = regexp.MustCompile(`<<(-?)\s*(?:'([^'\n]+)'|"([^"\n]+)"|\\?([^\s;&|()<>'"\\]+))`)
	shellFeeder  = regexp.MustCompile(`(?i)(^|[;&|(\s/\\'"])(?:(?:ba|z|da|k)?sh|pwsh|powershell|eval)(\.exe)?['"]?(\s|$|[<;&|)])`)
)

// stripHeredocs removes heredoc bodies. A commit message written as a heredoc is
// text, and its apostrophes would otherwise read as an open quote; the body of a
// heredoc fed to a shell is code, so that one is kept and read as commands.
func stripHeredocs(s string) string {
	if !strings.Contains(s, "<<") {
		return s
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		out = append(out, lines[i])
		m := heredocStart.FindStringSubmatchIndex(lines[i])
		if m == nil || (m[0] > 0 && strings.IndexByte("<\\", lines[i][m[0]-1]) >= 0) || midWordQuote(lines[i], m[1]) || shellFeeder.MatchString(lines[i]) ||
			inQuotes(priorText(out)+lines[i][:m[0]]) {
			continue
		}
		delim := firstNonEmpty(lines[i], m[4:])
		if isNumber(delim) {
			continue // `1 << 2` is arithmetic, not a heredoc
		}
		trim := m[3] > m[2] // "<<-" strips leading tabs from the terminator
		for i+1 < len(lines) {
			i++
			l := lines[i]
			if trim {
				l = strings.TrimLeft(l, "\t")
			}
			if strings.TrimRight(l, "\r") == delim {
				out = append(out, lines[i])
				break
			}
		}
	}
	return strings.Join(out, "\n")
}

func isNumber(s string) bool {
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

// midWordQuote reports a quote or backslash right after the delimiter the regexp
// read, as in `<<E\OF`: the shell unquotes it to a delimiter this code did not
// capture, so the body cannot be told apart from code and is left in.
func midWordQuote(line string, end int) bool {
	return end < len(line) && strings.IndexByte("\\'\"", line[end]) >= 0
}

// priorText is every line before the last one in out, each ended by a newline,
// so a quote or a comment opened on an earlier line is still open here.
func priorText(out []string) string {
	if len(out) < 2 {
		return ""
	}
	return strings.Join(out[:len(out)-1], "\n") + "\n"
}

// firstNonEmpty returns the text of the first participating capture group in idx
// (start,end pairs) of line.
func firstNonEmpty(line string, idx []int) string {
	for i := 0; i+1 < len(idx); i += 2 {
		if idx[i] >= 0 && idx[i+1] > idx[i] {
			return line[idx[i]:idx[i+1]]
		}
	}
	return ""
}

// inQuotes reports whether the end of prefix sits inside a quote that no
// `$(...)` has reopened, so a `<<` there is text, not a heredoc.
func inQuotes(prefix string) bool {
	var s quoteScan
	for i := 0; i < len(prefix); i++ {
		i = s.feed(prefix, i)
	}
	return s.q != 0 || s.comment
}

// quoteScan follows quoting through a shell text: the current quote, the quote
// to return to when a `$(` closes, and whether a comment is open.
type quoteScan struct {
	q       byte
	stack   []byte
	comment bool
}

// feed reads prefix[i] and returns the index of the last byte it consumed.
func (s *quoteScan) feed(prefix string, i int) int {
	c := prefix[i]
	switch {
	case s.comment:
		s.comment = c != '\n'
	case c == '#' && s.q == 0 && (i == 0 || strings.IndexByte(" \t\n;&|(", prefix[i-1]) >= 0):
		s.comment = true
	case c == '\\' && s.q != '\'':
		return i + 1
	case c == '$' && s.q != '\'' && i+1 < len(prefix) && prefix[i+1] == '(':
		s.stack = append(s.stack, s.q)
		s.q = 0
		return i + 1
	case c == ')' && len(s.stack) > 0:
		s.q = s.stack[len(s.stack)-1]
		s.stack = s.stack[:len(s.stack)-1]
	default:
		s.q = quoteAfter(s.q, c)
	}
	return i
}

// quoteAfter is the quote state once c has been read.
func quoteAfter(q, c byte) byte {
	switch {
	case q == '\'':
		if c == '\'' {
			return 0
		}
	case c == '"' && q == '"':
		return 0
	case (c == '"' || c == '\'') && q == 0:
		return c
	}
	return q
}
