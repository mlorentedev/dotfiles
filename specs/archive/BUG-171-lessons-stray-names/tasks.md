---
tags: [spec, tasks, templates]
created: "2026-11-12"
id: "BUG-171-lessons-stray-names"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-171-lessons-stray-names

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/3415-lessons-stray-names`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] [AC3] Failing test `TestLessonsFmt_CheckNamesAFileInAnotherNamingScheme`: a stray file at the root and one in a category directory are named under `--check`; `_format.md`, `README.md` at both levels and a `.txt` are not; `fmt` leaves the stray file in place and names it
- [x] [AC1] [AC2] `lessons.StrayNames`: the markdown files in `lessonDirs` that match neither `fileRE` nor the directory's own documents
- [x] [AC1] [AC3] Wire it into `fmt` and `fmt --check`, with a fix line that does not point at `renumber` (it refuses a file with no number)
- [x] Refactor: move fmt's reporting into `lessonsLeftAlone` and `reportLessonsCheck`, a table of findings, because a third kind pushed `RunE` past gocyclo

## Closing

- [x] `go test`, `go vet` and `golangci-lint run`, natively and with `GOOS=windows`
- [ ] `verification.md` filled in; independent review; archive

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/BUG-171-lessons-stray-names/features.json`):

```json
[
  {
    "id": "BUG-171-lessons-stray-names-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
