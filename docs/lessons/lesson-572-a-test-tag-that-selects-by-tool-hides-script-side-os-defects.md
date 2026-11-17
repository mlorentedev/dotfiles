---
id: "lesson-572-a-test-tag-that-selects-by-tool-hides-script-side-os-defects"
type: lesson
status: active
title: "A CI tier that selects tests by tool cannot see a script that is wrong per OS"
created: "2026-11-10"
---

# A CI tier that selects tests by tool cannot see a script that is wrong per OS

## Context

`tests/check-review-attestation.bats` gates the review-attestation check. It was
green on Linux and never ran on Windows, because the Windows CI step selects
`--filter-tags windows` over a fixed file list and this file was not on it.

The tag's meaning was "the test runs `pwsh`", and by that meaning the file was
correctly excluded: its 62 tests are shell and jq, nothing PowerShell. The script
under test, though, reads jq output through `IFS=$'\t' read`. On Windows
`jq.exe` terminates every line with CRLF, so the CR lands on the last field of
each record — `marker` arrives as `## PR Reviewer Guide\r`, `advisory` as
`true\r`. Neither ever matches. Twelve of the 62 tests were red on a clean tree;
nine of them for this reason.

## Problem

The tier answered "which tests need Windows?" and the question that mattered was
"which *scripts* behave differently on Windows?". Those are different sets, and
the second one is not derivable from the first: a test can be perfectly
OS-neutral while the script it exercises is not.

Two consequences made it durable rather than a one-off:

- **The defect fails closed.** A CR can only make a marker match harder, never
  easier, so the gate kept refusing. What changed was the *recorded reason*:
  `attested` degraded to `exempt`, and an advisory decline was handled as a hard
  one — the distinction #1045 introduced, lost. Both paths exit 0, so the exit
  code the CI step reports said nothing.
- **The class was already a lesson.** lesson-113 documents exactly this: `$(jq …)`
  strips the CRLF, `< <(jq …)` plus `read` does not. The fix had been made once,
  locally, in `scripts/check-retired-skills.sh:42`, and never generalised. The
  second and third sites sat in a file nobody ran on Windows.

## Solution

Strip at the producer, both sites (`| tr -d '\r'`), following the precedent
rather than inventing per-field `${marker%$'\r'}` handling, which misses the
next field someone adds.

For the tier: widen what the tag means and say so where the guard lives.
`tests/ci-windows-bats-tier.bats` now records that `windows` marks a case whose
behaviour is correct on Linux and wrong on Windows *for a reason Linux cannot
exhibit* — a jq-produced field carrying CRLF, a path handed to a native binary —
not only a case that runs `pwsh`. Four cases carry it rather than all nine OS
sensitive ones: loading that file costs ~73 s of bats startup on Windows before
a single test runs, so the tag buys coverage per second, one case per root cause.
Linux still runs all 62.

## Rule

A per-OS CI tier is a statement about *coverage*, and it is only as good as the
predicate that selects. Selecting by "which tool does this test call" answers a
question about the test; portability defects live in the script the test
exercises. When a gate's suite is excluded from an OS, the script it guards is
unverified on that OS — and if the failure mode is "refuses, but for the wrong
recorded reason", the exit code will not tell you. Ask what the tag *means*
before trusting what it selects, and when a fix has a precedent in the same repo,
grep for the other sites in the same commit.
