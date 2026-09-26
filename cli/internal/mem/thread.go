package mem

import (
	"fmt"
	"regexp"
	"strings"
)

// ThreadLabels are a handoff thread's fields, in the order a thread is written
// (MEMORY-008). They are defined once, here, so the writer, the resume renderer
// and the skills that ask for these fields cannot disagree on a name.
//
// `Verify` holds the commands that confirm the state the thread describes, and
// `Awaiting Manu` the decisions only he can take, so "waiting on a decision" is
// a next action a session can report instead of inventing work.
var ThreadLabels = []string{
	"Updated", "Last task", "Decisions", "Open threads", "Next action",
	"Verify", "Awaiting Manu", "Journal",
}

// threadLabelAliases are the labels sessions improvised for a canonical field
// before the set existed. They are read as the field they meant, so no existing
// block needs migrating by hand.
var threadLabelAliases = map[string]string{
	"Verify at start":          "Verify",
	"Judgment calls left open": "Awaiting Manu",
}

// Field is one labelled part of a thread.
type Field struct {
	// Label is a canonical label, the block's own label when it is outside the
	// set, or "" for text before the first label.
	Label string
	Value string
}

// Thread is a handoff thread read back into its fields.
type Thread struct {
	Key    string // from the heading; "" when the block has none
	Fields []Field
}

// boldLabel matches `**Label:** value`, optionally blockquoted.
var boldLabel = regexp.MustCompile(`^\s*(?:>\s*)?\*\*([^*]+?):\*\*\s*(.*)$`)

// plainLabel matches the two fields sessions write without bold. Any other
// `Word:` at the start of a line is prose, so only these two are read.
var plainLabel = regexp.MustCompile(`^\s*(?:>\s*)?(Updated|Journal):\s*(.*)$`)

// ParseThread reads a thread block, with or without its heading, into fields.
//
// No line of the body is dropped. A label outside the set is kept as written,
// and so is a qualified canonical one (`Decisions (Manu, 2026-09-24)`), because
// the qualifier is content. Text before the first label is kept under the empty
// label, and a line that is not a label belongs to the field above it, so an
// ordinary `###` subheading in a body is content, as it is for WriteThread.
func ParseThread(block string) Thread {
	var th Thread
	lines := strings.Split(strings.ReplaceAll(block, "\r\n", "\n"), "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	if len(lines) > 0 {
		if key, ok := threadHeadingKey(lines[0]); ok {
			th.Key, lines = key, lines[1:]
		}
	}

	var label string
	var value []string
	started := false
	flush := func() {
		v := strings.Join(trimBlankEdges(value), "\n")
		if started || v != "" {
			th.Fields = append(th.Fields, Field{Label: label, Value: v})
		}
	}
	for _, line := range lines {
		l, v, ok := fieldLine(line)
		if !ok {
			value = append(value, strings.TrimRight(line, " \t"))
			continue
		}
		flush()
		label, value, started = l, []string{v}, true
	}
	flush()
	return th
}

// fieldLine reports whether line opens a field, and its label and inline value.
func fieldLine(line string) (string, string, bool) {
	m := boldLabel.FindStringSubmatch(line)
	if m == nil {
		m = plainLabel.FindStringSubmatch(line)
	}
	if m == nil {
		return "", "", false
	}
	label := strings.TrimSpace(m[1])
	if canonical, ok := threadLabelAliases[label]; ok {
		label = canonical
	}
	return label, strings.TrimRight(m[2], " \t"), true
}

// ThreadWarnings names what a handoff body lacks, or carries outside the
// canonical set, for handoff-write to print. A warning never blocks the write: a
// handoff in an odd shape is still a handoff, and refusing it would lose it.
func ThreadWarnings(body string) []string {
	th := ParseThread(body)
	var out []string
	if v, ok := th.Get("Next action"); !ok || strings.TrimSpace(v) == "" {
		out = append(out, "no Next action, so the next session has no first step")
	}
	for _, f := range th.Unknown() {
		out = append(out, fmt.Sprintf("label %q is not one of: %s", f.Label, strings.Join(ThreadLabels, ", ")))
	}
	return out
}

// Get returns the value under a canonical label, and whether the thread has it.
func (t Thread) Get(label string) (string, bool) {
	for _, f := range t.Fields {
		if f.Label == label {
			return f.Value, true
		}
	}
	return "", false
}

// Unknown returns the labelled fields outside the canonical set, in order.
func (t Thread) Unknown() []Field {
	var out []Field
	for _, f := range t.Fields {
		if f.Label != "" && !isThreadLabel(f.Label) {
			out = append(out, f)
		}
	}
	return out
}

func isThreadLabel(label string) bool {
	for _, l := range ThreadLabels {
		if l == label {
			return true
		}
	}
	return false
}
