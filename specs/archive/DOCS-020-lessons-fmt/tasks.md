---
tags: [spec, tasks, templates]
created: "2026-10-06"
---

# Tasks - DOCS-020-lessons-fmt

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/docs-020-lessons-fmt`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

### PR 1 — the format, the generator and the migration (this PR)

- [x] [AC1] Failing tests over the three measured shapes plus a blank-line and a bare-number H1, idempotence asserted on each; `Normalize` implemented. An H1 pattern ending in `\s*` ate the blank lines after it, one per run; the command-level test caught it and `[ \t]*` fixed it
- [x] [AC2] `RenderIndex`: the hand-kept table is replaced once, then the markers; prose kept; `|` escaped; rendering a generated index changes nothing
- [x] [AC3] `Plan`: a duplicate number, lessons without an index, and lessons in a category without the root index are errors
- [x] [AC4] `dotf lessons fmt [--check] [--dir]`
- [x] Migration: `fmt` on the repository (339 lessons and the index); a second `fmt --check` passes; `check-lessons.sh` still passes
- [x] [AC5] The `dotf init` index template is a fixed point of `RenderIndex`
- [x] [AC6] `repo-hygiene.yml` runs `go run ./cmd/dotf lessons fmt --check`
- [x] Vault: skills and templates updated (`02192bb9`); harness skill records refreshed; `compile-harness.sh --check` clean

### PR 2 — after the release that carries `dotf lessons` is the `DOTF_VERSION` pin

- [x] [AC7] Failing test, then `Plan` refuses a wikilink that names no lesson. The task below assumed every case of the
      deleted bats already lived in Go; the wikilink assertion of `guard-lesson-numbers-unique.bats` did not
- [x] [AC7] Pre-commit runs `dotf lessons fmt --check` instead of `check-lessons.sh` (DOTF_VERSION 0.65.0 carries `lessons`)
- [x] [AC7] Delete `scripts/check-lessons.sh`, `tests/check-lessons.bats` and `tests/guard-lesson-numbers-unique.bats`, and
      the hygiene workflow's shell step

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/DOCS-020-lessons-fmt/features.json`):

```json
[
  {
    "id": "DOCS-020-lessons-fmt-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
