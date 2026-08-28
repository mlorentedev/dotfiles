# Lesson 241 — Re-running a command just to print what it already told you inherits its exit status, silently, under `pipefail`

**Date:** 2026-08-27
**Area:** shell / `set -euo pipefail` / vault-health.sh
**Severity:** medium — a real run silently drops every backlog file after the first drifted one, the advisory merged-check pass, and the closing report

## What happened

Found while characterizing `scripts/vault-health.sh` for its Go port (CLI-021
increment 2, #490). One golden fixture — a task file with a real duplicate-id
drift — captured an `expected/stdout` that stops mid-section, with no closing
`========================================` banner and no `Results: N passed...`
line at all. That looked like a broken capture. It was the shell's actual
behaviour.

Section 7 checks each backlog file twice: once to decide pass/fail, once more
to print the failure's detail:

```sh
if ! "$SCRIPT_DIR/check-backlog-integrity.sh" "$tasks" >/dev/null 2>&1; then
    fail "Backlog drift in ..."
    "$SCRIPT_DIR/check-backlog-integrity.sh" "$tasks" 2>/dev/null | sed 's|^|        |'
fi
```

The first invocation is negated (`! cmd`), which [Lesson
224](lesson-224-a-negated-assertion-is-exempt-from-set-e-so-it-cann.md) already
covers: exempt from `set -e`, safe. The second invocation is not negated, and
its whole purpose is to be re-run only for its stdout — the exit status is
supposed to be irrelevant. But it is piped into `sed`, and under `pipefail` the
pipeline's status is the *script's* 1, not `sed`'s 0. `set -e` sees an
unguarded, non-zero pipeline and kills the whole script right there: no later
file in the same loop, no second (merged-check) loop, no footer.

## Why it survives inspection

**The sibling loop two paragraphs down does the right thing, which hides the
bug in the one above it.** The merged-check pass captures first —
`merged_out="$(check-backlog-merged.sh "$tasks" 2>/dev/null)"` inside its own
negated test — then prints the captured *string* through `printf | sed`. A
string can't fail. Reading the file top-to-bottom, the second loop looks like
confirmation that the pattern is fine, when it is actually the fix the first
loop is missing.

**The failure mode is invisible from a shell prompt.** Run the script
interactively against a single bad file and it looks correct: one `FAIL` line,
one detail block, exit 1. The gap only appears with a *second* backlog file
after the first bad one, or with the merged-check advisory that never gets a
chance to run — both of which require multiple drifted files or a specific
task-file layout to notice by eye.

**A stub-backed characterization test caught it where reading the script did
not.** The stub obsidian binary made every OTHER section's obsidian-dependent
output deterministic and boring, which is precisely what let the ONE real
external process in the fixture (`check-backlog-integrity.sh`, run for real,
not stubbed — SDD-012's own text-parsing checks have no GUI dependency to stub)
stand out when its exit code silently ended the run.

## The rule

**Never re-run a command a second time solely to print output you could have
captured the first time.** If a command's own exit status is meant to be
irrelevant at a given call site, capture its stdout into a variable instead of
re-invoking it — `printf '%s\n' "$captured" | sed ...` then pipes an
already-materialized STRING, whose own success has nothing to do with what
produced it, so that pipeline cannot inherit the original command's exit
status. Re-executing is not "the same thing done twice": under
`set -e -o pipefail`, the second invocation is a fresh, unguarded command whose
own non-zero exit can end the script, this time with no `!` standing in front
of it.

## Not fixed at the source

This is an ORACLE defect, found while building a Go characterization corpus
that must prove equivalence to what `vault-health.sh` does TODAY, not what it
should do. Reproduced faithfully in the port
(`cli/internal/vault/health.go`'s `section7Backlog`, pinned by the
`backlog-drift` golden) rather than "corrected while translating" — the same
discipline [Lesson
223](lesson-223-a-test-updated-to-keep-passing-stops-being-a-guar.md) argues
for test assertions applies to a characterization oracle. The actual shell fix
— capture-then-print, mirroring the merged-check loop already next to it — is
ticketed separately: #1314.

## Evidence

- `tests/golden/vault-health/cases/backlog-drift/expected/stdout` — captured
  from the real shell, ends after the first file's detail block, no footer
- `check-backlog-integrity.sh` exit-code contract: 1 for drift/contradiction —
  documents the value the pipeline inherits
- `tests/vault-health-go-parity.bats` — 16/16 byte-identical including this case
- `#1314` — the ticket for the actual shell fix
