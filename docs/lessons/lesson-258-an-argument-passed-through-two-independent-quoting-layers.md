# Lesson 258 — An argument passed through two independent quoting conventions is only as safe as the weaker one

**Date:** 2026-09-02
**Context:** HARNESS-050 (#575) — `memlink.createLink`'s Windows junction creation
failing on a path containing a bare comma

## What happened

`createLink`'s Windows branch ran `exec.Command("cmd", "/c", "mklink", "/J", target, src)`.
Go's `os/exec` escapes argv elements into a single command-line string on Windows following
the C-runtime/`CommandLineToArgvW` convention: quote an argument only if it contains a space,
tab, or embedded quote. A path like `it,_and` has none of those, so it passed through
unquoted.

`mklink` is a `cmd.exe` **builtin**, not a separate executable — so there is no second
argv-parsing layer downstream that would re-split it. Only `cmd.exe`'s own tokenizer ever
sees the command tail, and that tokenizer treats a bare comma, semicolon, or equals sign as a
word separator outside quotes — a rule the C-runtime convention knows nothing about. The
unquoted comma split `mklink`'s argument list, and it failed "the syntax of the command is
incorrect": a silent no-op, since the caller (`Ensure`) swallows the error by contract.

Spaces happened to round-trip. That was coincidence, not evidence the quoting was correct —
both conventions quote on space, so the common case masked the defect for as long as no path
with a delimiter-but-no-space was tried.

## The fix

Two quoting conventions apply to this one command line, and only one of them is under the
caller's control by default. The fix takes control of both:

1. Bypass Go's automatic escaping entirely via `syscall.SysProcAttr{CmdLine: ...}` — this
   string is used verbatim as the process's command line, so Go's argv-to-string conversion
   never runs.
2. Quote both paths for `cmd.exe`'s tokenizer directly, and invoke with `cmd /s /c "mklink
   /J "<link>" "<src>""`. The `/S` switch is what makes this work: it tells `cmd` to strip
   only the outermost quote pair from the tail and hand the still-quoted
   `mklink /J "..." "..."` straight to the builtin's own argument scanner, instead of
   `cmd`'s default of treating one quoted tail as a single opaque token.

Verified empirically against a real `cmd.exe` (a throwaway Go probe, not just reasoning about
quoting rules) before landing the shape — comma, semicolon, paren, equals, and space path
components all round-trip.

## The lesson

**When an argument crosses two independent parsers on its way to the thing that finally
reads it, being safe under one convention says nothing about the other.** `os/exec`'s
escaping and `cmd.exe`'s tokenizer are both real, both documented, and disagree about which
characters are delimiters — and the failure mode (a silently swallowed error, per this
caller's contract) gave no signal that the second layer even existed until a path shaped the
right way was tried.

Two corollaries:

- **A cmd.exe builtin (`mklink`, `dir`, `copy`, ...) has no second argv layer to quote
  through — only `cmd`'s own tokenizer parses it, so quoting has to satisfy *that* parser
  specifically, not the generic "does this need quotes" convention `os/exec` applies by
  default.
- **"It works for the paths I tried" is not evidence a quoting fix is correct** when the
  parser doing the splitting is undocumented in the code and only informally understood.
  Verify against the real interpreter (a throwaway probe is cheap) rather than trust
  recollected quoting rules — the difference between "likely correct" and "verified correct"
  here was five minutes with a scratch Go file.

## Where it bit

`cli/internal/memlink/memlink_windows.go` (`createLink`, new — split out of the shared
`memlink.go` specifically because this quoting logic doesn't belong in a file compiled on
every OS), `cli/internal/memlink/memlink_test.go`
(`TestCreateLink_CmdDelimiterPaths`).
