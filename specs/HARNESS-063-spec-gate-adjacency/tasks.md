---
tags: [spec, tasks, templates]
created: "2026-08-09"
---

# Tasks - HARNESS-063-spec-gate-adjacency

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers**: `[P]` = no dependency on another unchecked task; `[AC<n>]` = satisfies acceptance criterion `<n>` from `proposal.md`.

## Setup

- [x] Branch created from main: `feat/spec-gate-issue-adjacency`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

TDD order. The #851 red-test comes first because it is the acceptance criterion
that decides whether the whole direction was worth choosing (#858 AC3).

- [x] [P] [AC2] Add the `#849` fixture — real title + body, inline code span intact —
      and a failing bats case replaying #851's changed-file list against it
- [x] [AC1] [AC2] Implement `_adjacent_open_issues` in `check-spec-gate.sh`:
      match changed-file paths and basenames over **unstripped** title + body
- [x] [AC1] Add `--adjacency-issues <file>` parsing + the advisory report writer
      (`::warning::` annotation + `$GITHUB_STEP_SUMMARY` table)
- [x] [P] [AC3] Test: a PR with adjacent issues exits with the same status as
      without them — the report cannot change the verdict
- [x] [P] [AC4] Test: no flag and no token ⇒ output byte-identical to the current
      version (guards the offline pre-push path, #854)
- [x] Refactor pass: keep `_adjacent_open_issues` under the 40-line function limit,
      no reuse of `_strip_markdown_code` (see proposal Risks). The matcher is split;
      `_report_adjacent_issues` (39 lines) splitting is #2303.
- [x] [AC1] Wire `spec-gate.yml`: `permissions: issues: read`, one `gh issue list`
      fetch into a file, pass `--adjacency-issues`. The fetch shipped in #860 and the
      flag only in #885 (`327feac2`); the archive PR makes the wiring pin step-scoped
      and announces a feed truncated at the limit.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `shellcheck scripts/check-spec-gate.sh` passes
- [x] `bats tests/*.bats` passes (no regression in the existing spec-gate suite)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in with output produced this session
- [x] PR opened referencing this spec folder, body carries `Refs #858` — **not**
      a closing keyword: #858 closes with the fixture-shape inventory in #857
