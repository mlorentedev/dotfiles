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
}

// command is one simple command: its words in order.
type command []word

// frame remembers the quote state to return to when a `$(`/backtick closes.
type frame struct {
	quote    byte
	backtick bool
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
	q     byte // 0, '\'' or '"'
}

// splitCommands breaks a shell string into simple commands. It reads `&&`, `||`,
// `;`, `|`, `&`, newlines, subshell parentheses, `$(...)` and backticks as
// boundaries, and never evaluates anything. ok is false when quoting or a
// substitution is left open: the commands are then best-effort.
func splitCommands(s string, shell Shell) (cmds []command, ok bool) {
	lx := &lexer{shell: shell}
	rs := []rune(stripHeredocs(s))
	for i := 0; i < len(rs); i++ {
		i = lx.step(rs, i)
	}
	lx.endCommand()
	return lx.cmds, lx.q == 0 && len(lx.stack) == 0
}

func (lx *lexer) endWord() {
	if lx.has {
		lx.cur = append(lx.cur, word{text: lx.text.String(), raw: lx.raw.String(), dynamic: lx.dyn, quoted: lx.quote})
	}
	lx.text.Reset()
	lx.raw.Reset()
	lx.has, lx.dyn, lx.quote = false, false, false
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
	if lx.q == '\'' {
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
	case c == '$' && i+1 < len(rs) && rs[i+1] == '(':
		lx.endCommand()
		lx.stack = append(lx.stack, frame{quote: lx.q})
		lx.q = 0
		return i + 1, true
	case c == '`' && lx.shell == ShellPOSIX:
		lx.endCommand()
		if n := len(lx.stack); n > 0 && lx.stack[n-1].backtick {
			lx.q = lx.stack[n-1].quote
			lx.stack = lx.stack[:n-1]
		} else {
			lx.stack = append(lx.stack, frame{quote: lx.q, backtick: true})
			lx.q = 0
		}
		return i, true
	case c == '$':
		lx.dyn = true
		lx.put(c)
		return i, true
	case c == '\\' && lx.shell == ShellPOSIX:
		return lx.backslash(rs, i), true
	case c == '`' && lx.shell == ShellPowerShell && i+1 < len(rs):
		lx.put(rs[i+1])
		return i + 1, true
	}
	return i, false
}

func (lx *lexer) backslash(rs []rune, i int) int {
	if i+1 >= len(rs) {
		lx.put('\\')
		return i
	}
	n := rs[i+1]
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
	case c == '\'':
		lx.q, lx.has, lx.quote = '\'', true, true
	case c == '"':
		lx.q, lx.has, lx.quote = '"', true, true
	case c == ' ' || c == '\t' || c == '\r':
		lx.endWord()
	case c == '#' && !lx.has:
		for i+1 < len(rs) && rs[i+1] != '\n' {
			i++
		}
	case c == '&' && lx.redirectAmp(rs, i):
		lx.put(c)
	case strings.ContainsRune(";&|\n()", c), lx.shell == ShellPowerShell && (c == '{' || c == '}'):
		lx.boundary(c)
	default:
		lx.put(c)
	}
	return i
}

// boundary ends the current command. A `)` that closes a `$(` also restores
// the quote state the substitution was opened in.
func (lx *lexer) boundary(c rune) {
	lx.endCommand()
	if c != ')' {
		return
	}
	if n := len(lx.stack); n > 0 && !lx.stack[n-1].backtick {
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
		if w.quoted {
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
	for _, op := range []string{"&>>", "&>", ">>", ">&", ">|", "<<<", "<<", "<&", ">", "<"} {
		if rest, ok := strings.CutPrefix(s, op); ok {
			if op == ">&" || op == "<&" {
				return "x", true // `2>&1`: the target is part of the word
			}
			return rest, true
		}
	}
	return "", false
}

var (
	heredocStart = regexp.MustCompile(`<<(-?)\s*(?:'([^'\n]+)'|"([^"\n]+)"|\\?([A-Za-z_][A-Za-z0-9_]*))`)
	shellFeeder  = regexp.MustCompile(`(^|[;&|(\s])(?:ba|z|da|k)?sh(\s|$)|(^|[;&|(\s])eval(\s|$)`)
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
		if m == nil || shellFeeder.MatchString(lines[i][:m[0]]) {
			continue
		}
		delim := firstNonEmpty(lines[i], m[4:])
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
