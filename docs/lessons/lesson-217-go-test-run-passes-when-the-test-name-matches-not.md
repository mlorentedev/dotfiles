# Lesson 217 — `go test -run` passes when the test name matches nothing

**Date:** 2026-08-21
**Context:** Writing the `features.json` verification contract for HARNESS-075.

## What happened

Five of eight machine-readable acceptance criteria were verified with commands shaped like:

```bash
cd cli && go test ./internal/harness/ -run TestModelMapValidatesAgainstSchema
```

Every one exited 0. They also exit 0 when the test does not exist:

```
$ go test ./internal/harness/ -run TestThisNameDoesNotExistAnywhere
ok  github.com/mlorentedev/dotfiles/cli/internal/harness  0.002s [no tests to run]
$ echo $?
0
```

So the moment any of those tests were renamed or deleted, its criterion would keep reporting PASS.
What the commands actually asserted was **that the package compiles**.

## The fix

Require the test to have reported a pass. This fails on a *missing* test and on a *failing* one:

```bash
cd cli && go test ./internal/<pkg>/ -run '^<Name>$' -v 2>&1 | grep -q -- '--- PASS: <Name> '
```

| state | exit |
|---|---|
| test exists and passes | 0 |
| name matches nothing | 1 |
| test exists and fails | 1 |

Here the pipeline's exit code belonging to `grep` is the **point**, not the hazard.

**The first attempt at this fix was also wrong**, and cheaply: anchoring as `'--- PASS: <Name>$'`
never matched, because `go test -v` appends a duration — `--- PASS: TestX (0.00s)`. That version
failed closed rather than open, so it surfaced on the first run. Worth noting anyway: a check that
always fails gets disabled rather than fixed, which is its own way of losing a guard.

## The lesson

**A verification command must fail when the thing it verifies is absent, not only when it is
broken.** Those are different failures and most tooling conflates them by exiting 0 for "nothing
matched". `go test -run` is one instance; `grep` without `-q` in a conditional, a linter pointed at
a path that does not exist, and a test suite whose glob matched no files are others.

The tell is a success message containing a phrase like *"no tests to run"*, *"0 files checked"*, or
*"nothing to do"*. **A tool that reports what it did not do, and exits 0 anyway, is reporting health
it never established.**

This one is worth naming separately from its neighbours because of *where* it was found: inside the
`features.json` of a spec whose entire purpose is a routing map that must fail loudly rather than
resolve to a permissive default. The spec written to enforce that constraint violated it three
times in its own verification block — twice with commands that could never pass, once with five
that could never fail.

## See also

- `docs/lessons/lesson-215-a-parser-for-one-runner-reads-the-other-runners-re.md` — the same class
  at the parsing layer: a complete review read as zero characters
- `docs/lessons/lesson-212-an-invalid-instrument-is-indistinguishable-from-an.md`
- `.claude/CLAUDE.md` — the prohibited-pattern table's warning about silently-empty results
