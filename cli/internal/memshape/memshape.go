// Package memshape detects and repairs auto-memory MEMORY.md files whose body
// was wrapped inside a YAML block scalar (`content: |`).
//
// That shape is invalid state, not a supported variant: the vault's scaffolding
// SSOT (00_meta/templates/agent-memory.md) forbids it outright, nothing dotfiles
// ships emits it, and every affected file was produced by a single bulk edit on
// 2026-05-26 (vault commit 1c216229) with nothing wrapping a file since. It is
// repaired rather than supported because crystallize anchors every marker at
// column 0, so a wrapped file silently fails to crystallize (#857), and because
// no YAML emitter can round-trip it — a literal block scalar cannot carry
// trailing whitespace, so re-emitting strips the markdown hard breaks the
// handoff convention depends on. See #864 and
// specs/CLI-021-dotf-vault-build-knowledge/evidence-yaml-roundtrip.md.
package memshape

import (
	"errors"
	"regexp"
	"strings"
)

// blockOpener matches a block-scalar key at column 0: `content: |`, tolerating
// the chomping/indentation indicators YAML allows (`|-`, `|+`, `|2`).
//
// Deliberately anchored at column 0 and NOT keyed to the name "content": the
// detection must describe the SHAPE, not the one key this machine happens to
// use, or it silently passes a file wrapped under a different name.
var blockOpener = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*):[ \t]*\|[-+0-9]*[ \t]*$`)

// ErrNotWrapped is returned by Unwrap for a file that is not a wrapped
// MEMORY.md. Callers treat it as "nothing to do", never as a failure.
var ErrNotWrapped = errors.New("memshape: not a YAML-wrapped MEMORY.md")

// IsWrapped reports whether src is a MEMORY.md whose body sits inside a YAML
// block scalar: a `---` first line plus a block-scalar opener.
//
// Structural, never an indent probe. The indent is not a constant — it is
// derived per file in Unwrap — so nothing here may key on a literal width.
func IsWrapped(src string) bool {
	lines := strings.Split(src, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != "---" {
		return false
	}
	for _, l := range lines[1:] {
		if blockOpener.MatchString(l) {
			return true
		}
	}
	return false
}

// leadingSpaces counts the leading space characters of l.
func leadingSpaces(l string) int {
	for i := 0; i < len(l); i++ {
		if l[i] != ' ' {
			return i
		}
	}
	return len(l)
}

// blank reports whether l holds no content — either empty or whitespace only.
// Such lines are excluded from indent measurement: a truly empty line has zero
// indent and would drag any minimum to zero, and a whitespace-only line inside
// a block scalar is padding, not content.
func blank(l string) bool { return strings.TrimSpace(l) == "" }

// Unwrap converts a YAML-wrapped MEMORY.md back to plain markdown: the
// frontmatter is preserved verbatim minus the block-scalar key, closed with
// `---`, and the body is de-indented and emitted after it.
//
// The de-indent is the whole subtlety, and getting it "right" by YAML's rule
// alone is not enough. YAML sets a block scalar's indentation from its first
// non-empty line. But the 2026-05-26 wrapper emitted the first body line one
// level shallower than every following line (4 vs 6) in 16 of the 17 affected
// files, so a faithful decode leaves the whole body except its first line
// carrying a residual indent — and crystallize's markers, anchored at column 0,
// still would not match. A migration that stopped at the YAML-correct answer
// would produce a file that parses cleanly and still fails to crystallize,
// which is the very defect it exists to remove.
//
// So two widths are removed, and BOTH are derived from the file:
//
//	blockIndent — the first non-empty body line's indent (YAML's own rule)
//	residual    — how much deeper every OTHER non-empty line sits
//
// residual is a minimum over the remaining non-empty lines, so a single line at
// blockIndent forces it to 0 and nothing is over-stripped. It is non-zero only
// when every line after the first is deeper, which is precisely the wrapper's
// signature. hive — the one file wrapped with a uniform indent — exercises the
// residual==0 path and is the regression case against over-stripping.
func Unwrap(src string) (string, error) {
	if !IsWrapped(src) {
		return "", ErrNotWrapped
	}

	lines := strings.Split(src, "\n")

	opener := -1
	for i, l := range lines {
		if blockOpener.MatchString(l) {
			opener = i
			break
		}
	}
	if opener < 0 {
		return "", ErrNotWrapped
	}

	// Everything between the leading `---` and the opener is frontmatter to keep.
	// A key appearing at column 0 *after* the opener would mean the block ends
	// early and this file is a shape we have never seen; refuse rather than guess.
	body := lines[opener+1:]
	for _, l := range body {
		if !blank(l) && leadingSpaces(l) == 0 {
			return "", errors.New("memshape: content found at column 0 after the block opener — unrecognised shape, refusing to guess")
		}
	}

	first := -1
	for i, l := range body {
		if !blank(l) {
			first = i
			break
		}
	}
	if first < 0 {
		return "", errors.New("memshape: block scalar has no content")
	}

	blockIndent := leadingSpaces(body[first])

	residual := -1
	for _, l := range body[first+1:] {
		if blank(l) {
			continue
		}
		if d := leadingSpaces(l) - blockIndent; residual < 0 || d < residual {
			residual = d
		}
	}
	if residual < 0 {
		residual = 0 // single-line body: nothing to compare against
	}

	var out strings.Builder
	for _, l := range lines[:opener] {
		out.WriteString(l)
		out.WriteString("\n")
	}
	out.WriteString("---\n\n")

	// Trailing whitespace is never touched: two trailing spaces are the markdown
	// hard break the handoff convention requires, and they are content here.
	for i, l := range body {
		strip := blockIndent + residual
		if i == first {
			strip = blockIndent
		}
		if n := leadingSpaces(l); n < strip {
			strip = n
		}
		out.WriteString(l[strip:])
		if i < len(body)-1 {
			out.WriteString("\n")
		}
	}

	return out.String(), nil
}
