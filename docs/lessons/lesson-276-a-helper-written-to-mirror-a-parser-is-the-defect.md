---
id: lesson-276
type: lesson
status: active
created: "2026-09-05"
owner: manu
tags: [lesson, go, parsing, duplication, unicode, build]
---

# 276 — A helper written to "mirror" an existing parser is the defect; the transcription bug is only how you find out

## What happened

HARNESS-120 needed a persona record's markdown BODY. The loader already had
`frontmatterBlock`, which splits the same file to get its YAML HEAD. Rather than
widen it, a new `splitRecordBody` was written alongside — with a comment that
named the hazard out loud:

```go
// splitRecordBody returns everything after the closing frontmatter fence. It
// mirrors frontmatterBlock's fence handling — including the CRLF a record
// deployed on Windows carries — because two different ideas of where a record
// ends is how one of them starts being wrong.
```

The transcription introduced a bug the author could not see. `frontmatterBlock`
strips a byte-order mark with the escape `"\ufeff"` — six ASCII characters. The
mirror was written with a **literal U+FEFF** pasted into the string instead:

```go
s := strings.TrimPrefix(string(raw), "\ufeff")   // loader: fine
s := strings.TrimPrefix(string(raw), "<U+FEFF>") // mirror: hard compile error
```

Go permits a BOM only as byte 0 of a source file. Anywhere else it is an illegal
identifier character, so this is not a runtime bug — it fails the build of the
**whole package**:

```
internal/harness/preamble.go:103:40: invalid BOM in the middle of the file
```

`go build`, `go vet` and `go test` were all red on `internal/harness` for an
entire session, and the character is invisible in the editor, in `git diff`, and
in code review.

## Why it happened

Two failures stacked, and only the second one is interesting.

**The visible one:** a zero-width character survives copy-paste and renders as
nothing. `cat -A` is the cheapest way to tell the escape from the byte:

```
$ grep -n TrimPrefix persona.go | cat -A
154:^Is := strings.TrimPrefix(string(raw), "\ufeff")$   <- six ASCII chars: correct
```

**The one worth keeping:** the comment justifying the duplicate was *correct
about the risk and drew the wrong conclusion from it*. "Two ideas of where a
record ends is how one of them starts being wrong" is an argument for having one
splitter, not for carefully synchronising two. The author reasoned all the way to
the hazard and then built it anyway, because the alternative — changing the
loader's signature — looked like the more invasive change.

It is the same shape as `check-roster-consistency.py` reporting "no skills" in
silence: a second reader of a format, written to match the first, that stopped
matching. That one failed *quietly* and shipped. This one failed loudly and cost
a session. The loud failure was the lucky outcome.

## The fix

Delete the mirror. Widen the original to return both halves, so there is exactly
one place that knows where a record ends:

```go
func splitRecord(raw []byte) (front []byte, body string, err error) {
	s := strings.TrimPrefix(string(raw), "\ufeff")
	...
	return []byte(rest[:end]), strings.TrimSpace(after), nil
}
```

`LoadPersona` takes `front`, the preamble takes `body`. The BOM line disappears
with the duplicate rather than being patched inside it — the patch would have
fixed the symptom and kept the defect.

Then test the branch the shipped data cannot reach. No record in the repo has
CRLF fences on Linux, but the Windows CI leg compiles the same tree, and a body
split that kept the `---` of a CRLF fence would prepend three dashes to every
dispatched instruction **on that platform only**. So `splitRecord` is table-driven
over four inputs, written here with `<CR>` and `<LF>` standing in for the escapes
themselves:

| case | input | want front | want body |
|---|---|---|---|
| `lf` | `---<LF>name: x<LF>---<LF><LF># Mandate<LF>Do the thing.<LF>` | `<LF>name: x` | `# Mandate<LF>Do the thing.` |
| `crlf` | same with `<CR><LF>` throughout | `<LF>name: x<CR>` | `# Mandate<CR><LF>Do it.` |
| `bom` | a leading U+FEFF, then the `lf` case | `<LF>name: x` | `body` |
| `empty body` | fences with nothing after them | `<LF>name: x` | *(empty)* |

Plus two malformed inputs — no fence, and an unclosed fence — which must each
return an **error and never an empty body**. An empty body would dispatch the
bare task, which is the generic agent the change exists to replace, behind a
successful exit.

> Written with `<LF>` rather than the real escapes on purpose: this repo's
> `check-md-escapes.sh` guard fails CI on a literal backslash-n followed by a
> line-start glyph, because that is the signature of a known markdown-corruption
> class. The first draft of this lesson tripped it. The guard was right and the
> prose changed — a lesson about invisible characters is a poor place to argue
> for an exception.

## The rule

**When you catch yourself writing a helper that "mirrors" an existing one, stop
and widen the original instead.** The duplication is the defect; whatever bug you
transcribe into the copy is only how you find out you made it. If the mirror
seems necessary because the original's signature is wrong, changing the signature
IS the smaller change — it is bounded by the compiler, while a silently diverging
parser is bounded by nothing.

Two corollaries:

- **A comment that names a hazard is not a mitigation of it.** If you can write
  down why the thing you are building is dangerous, you have already done the
  analysis that says not to build it.
- **`cat -A` before believing a string literal**, whenever the value contains
  anything invisible — BOMs, NBSPs, zero-width joiners. An editor showing nothing
  and a file containing nothing look identical.
