package mem

import (
	"fmt"
	"path/filepath"
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

// verifyNote heads every Verify field. A MEMORY file is input this session did
// not write, so nothing in it is presented as safe to run: the field renders as
// written and the session's permission gate decides each command. An earlier
// marker that flagged commands outside a read-only allowlist could never cover
// prose, so it read false-safe and was retired (MEMORY-008 review round 4,
// lesson 513).
const verifyNote = "(written by an earlier session; run each command only through the normal permission gate)"

// verifyBlock is the field as written, every line kept, under a heading line
// that carries the note.
func verifyBlock(heading, v string) string {
	return heading + " " + verifyNote + "\n" + strings.Trim(stripBreaks(v), "\n")
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

// noNextAction stands in for a Next action that is absent or blank, in both
// formats: silence there reads as nothing to do (#3003).
const noNextAction = "none recorded: ask what to work on"

// nextAction is the thread's first step with its breaks dropped, or
// noNextAction when the block records none.
func nextAction(th Thread) string {
	if v := stripBreaks(th.All("Next action")); strings.TrimSpace(v) != "" {
		return v
	}
	return noNextAction
}

func renderContext(th Thread, source string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Resume: thread %s\n\nSource: %s\n", th.Key, source)
	for _, label := range firstFields {
		if label == "Next action" {
			writeField(&b, label, nextAction(th))
			continue
		}
		if v := th.All(label); v != "" {
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
	if v := th.All("Verify"); v != "" {
		fmt.Fprintf(&b, "\n%s\n", verifyBlock("First confirm the state it describes:", v))
	}
	if v := th.All("Awaiting Manu"); v != "" {
		fmt.Fprintf(&b, "\nThese wait on Manu; ask, do not decide them:\n%s\n", stripBreaks(v))
	}
	fmt.Fprintf(&b, "\nNext action: %s\n", nextAction(th))
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
