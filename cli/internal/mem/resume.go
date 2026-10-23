package mem

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// SelectThread picks the thread a session resumes (MEMORY-008): agent's own fork
// of key when there is one, else key itself unless another agent wrote it, read
// by threadWriter as WriteThreadAs and session-end read it. A caller that names
// no agent gets only a thread nobody owns: it cannot show the work is its own.
// When neither holds, it chooses nothing and returns every thread key in order,
// so the caller lists them instead of handing a session somebody else's work.
func SelectThread(content, key, agent string) (string, []string) {
	keys := threadKeys(content)
	has := func(k string) bool {
		for _, x := range keys {
			if x == k {
				return true
			}
		}
		return false
	}
	if agent != "" && has(key+"+"+agent) {
		return key + "+" + agent, nil
	}
	if has(key) {
		if w := threadWriter(content, key, agent); w == "" || w == agent {
			return key, nil
		}
	}
	return "", keys
}

// threadKeys lists the section's thread keys in the order they appear.
func threadKeys(content string) []string {
	lines := strings.Split(content, "\n")
	start, end := handoffSection(lines)
	var keys []string
	for i := start + 1; start >= 0 && i < end; i++ {
		if k, ok := threadHeadingKey(lines[i]); ok {
			keys = append(keys, k)
		}
	}
	return keys
}

// ThreadText is key's full text and where it was read from: the thread file its
// entry points at, or memoryPath itself for an inline block (ADR-056).
func ThreadText(content, key, memoryPath string) (string, string) {
	memoryDir := filepath.Dir(memoryPath)
	lines := strings.Split(content, "\n")
	start, end := handoffSection(lines)
	s, e := threadSpan(lines, start, end, key)
	if s < 0 {
		return "", ""
	}
	block := strings.Join(trimBlankEdges(lines[s:e]), "\n")
	text := threadText(block, memoryDir)
	if m := pointerIn(block); m != "" && text != block {
		return text, filepath.Join(memoryDir, threadsDir, m)
	}
	return text, memoryPath
}

// pointerIn is the thread file a block's pointer names, or "".
func pointerIn(block string) string {
	for l := range strings.SplitSeq(block, "\n") {
		if m := threadPointer.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			return m[1]
		}
	}
	return ""
}

// readOnlyPrefixes are the Verify commands a resume presents as safe to run
// first. Anything else is marked unverified: a MEMORY file is input the next
// session did not write, and the session still runs every command through its
// own permission gate. Left off on purpose: generic file readers (cat, grep,
// head), which take any path, so `cat <secrets file>` must not render as safe;
// and subcommands with a writing form (git branch -D, git fetch, go build).
var readOnlyPrefixes = []string{
	"git status", "git log", "git diff", "git show", "git worktree list",
	"gh pr view", "gh pr checks", "gh pr list", "gh pr diff", "gh issue view", "gh issue list", "gh run view",
	"dotf spec audit", "dotf doctor", "dotf mem thread", "dotf mem resume", "dotf pr triage-queue",
	"go test", "go vet", "bats", "shellcheck",
}

// plainCommand is a command made only of characters the shell does not
// interpret, separated by plain spaces, so the shell splits it exactly where
// strings.Fields does: the check reads what will run. No quote, `$`, `~`,
// `=`, `:` or line break gets through, which is what defeated every narrower
// rule before it (${IFS}-exec=, a quoted '-exec=', HEAD:secrets).
var plainCommand = regexp.MustCompile(`^[A-Za-z0-9_./ -]+$`)

// readOnlyCommand is true only where read-only is decidable: a listed prefix
// followed by plain positional words that stay inside the checkout. Every
// flag is refused, because every escape hatch the reviews found was one
// (go test -exec, --outputdir, dotf doctor --fix, gh pr view --web); a
// command with flags renders unverified and still runs through the gate.
func readOnlyCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if !plainCommand.MatchString(cmd) {
		return false
	}
	for _, p := range readOnlyPrefixes {
		if cmd != p && !strings.HasPrefix(cmd, p+" ") {
			continue
		}
		for _, arg := range strings.Fields(strings.TrimPrefix(cmd, p)) {
			if strings.HasPrefix(arg, "-") || !insideCheckout(arg) {
				return false
			}
		}
		return true
	}
	return false
}

// insideCheckout is false for a path that can leave the working directory:
// absolute, or with a .. segment (./... is not one).
func insideCheckout(arg string) bool {
	if strings.HasPrefix(arg, "/") {
		return false
	}
	for _, seg := range strings.Split(arg, "/") {
		if seg == ".." {
			return false
		}
	}
	return true
}

// inlineCode finds the backticked commands in a Verify field.
var inlineCode = regexp.MustCompile("`([^`]+)`")

// unverifiedNote explains the in-place marker once per field.
const unverifiedNote = "(a command marked unverified is not a plain read-only one; run it only through the normal permission gate)"

// verifyText is a Verify field as written, every line kept, with each command
// that is not provably read-only marked where it stands. One rendering, so no
// unmarked copy of a marked command exists, and prose, caveats and bullets
// without a command all survive (ParseThread drops no line either).
func verifyText(v string) (string, bool) {
	marked := false
	out := inlineCode.ReplaceAllStringFunc(stripBreaks(v), func(span string) string {
		if readOnlyCommand(strings.Trim(span, "`")) {
			return span
		}
		marked = true
		return span + " (unverified)"
	})
	return strings.Trim(out, "\n"), marked
}

// verifyBlock is the field's text under a heading line, which carries the
// note when anything in it is marked.
func verifyBlock(heading, v string) string {
	text, marked := verifyText(v)
	if marked {
		heading += " " + unverifiedNote
	}
	return heading + "\n" + text
}

// RenderResume renders a thread's full text for the next session (MEMORY-008).
// It is a pure function of its arguments: the same thread renders the same
// bytes. "context" is the block a session-start hook injects; "prompt" is the
// paste-ready instruction a session used to type by hand.
func RenderResume(text, source, format string) (string, error) {
	th := ParseThread(text)
	switch format {
	case "context":
		return renderContext(th, source), nil
	case "prompt":
		return renderPrompt(th, source), nil
	}
	return "", fmt.Errorf("format %q: want context or prompt", format)
}

// firstFields are rendered first, in this order, because they decide what the
// session does before it reads anything else.
var firstFields = []string{"Next action", "Verify", "Awaiting Manu"}

func renderContext(th Thread, source string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Resume: thread %s\n\nSource: %s\n", th.Key, source)
	for _, label := range firstFields {
		if v, ok := th.Get(label); ok && strings.TrimSpace(v) != "" {
			writeField(&b, label, v)
		}
	}
	for _, f := range th.Fields {
		if f.Label == "" || isFirstField(f.Label) || strings.TrimSpace(f.Value) == "" {
			continue
		}
		writeField(&b, f.Label, f.Value)
	}
	return b.String()
}

func writeField(b *strings.Builder, label, v string) {
	switch label {
	case "Verify":
		fmt.Fprintf(b, "\n%s\n", verifyBlock("**Verify:**", v))
	default:
		sep := " "
		if strings.Contains(v, "\n") || strings.HasPrefix(strings.TrimSpace(v), "- ") {
			sep = "\n" // a list under the label, not glued to it, even of one item
		}
		fmt.Fprintf(b, "\n**%s:**%s%s\n", label, sep, stripBreaks(v))
	}
}

func isFirstField(label string) bool {
	for _, l := range firstFields {
		if l == label {
			return true
		}
	}
	return false
}

func renderPrompt(th Thread, source string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Resume thread %s. Its full handoff is %s; read it before acting.\n", th.Key, source)
	if v, ok := th.Get("Verify"); ok && strings.TrimSpace(v) != "" {
		fmt.Fprintf(&b, "\n%s\n", verifyBlock("First confirm the state it describes:", v))
	}
	if v, ok := th.Get("Awaiting Manu"); ok && strings.TrimSpace(v) != "" {
		fmt.Fprintf(&b, "\nThese wait on Manu; ask, do not decide them:\n%s\n", stripBreaks(v))
	}
	next := "none recorded: ask what to work on"
	if v, ok := th.Get("Next action"); ok && strings.TrimSpace(v) != "" {
		next = stripBreaks(v)
	}
	fmt.Fprintf(&b, "\nNext action: %s\n", next)
	return b.String()
}

// stripBreaks drops the `<br>` hard breaks a handoff carries for markdown.
func stripBreaks(v string) string {
	lines := strings.Split(v, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(strings.TrimSuffix(strings.TrimRight(l, " \t"), "<br>"), " \t")
	}
	return strings.Join(lines, "\n")
}
