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

- [ ] [P] [AC2] Add the `#849` fixture — real title + body, inline code span intact —
      and a failing bats case replaying #851's changed-file list against it
- [ ] [AC1] [AC2] Implement `_adjacent_open_issues` in `check-spec-gate.sh`:
      match changed-file paths and basenames over **unstripped** title + body
- [ ] [AC1] Add `--adjacency-issues <file>` parsing + the advisory report writer
      (`::warning::` annotation + `$GITHUB_STEP_SUMMARY` table)
- [ ] [P] [AC3] Test: a PR with adjacent issues exits with the same status as
      without them — the report cannot change the verdict
- [ ] [P] [AC4] Test: no flag and no token ⇒ output byte-identical to the current
      version (guards the offline pre-push path, #854)
- [ ] Refactor pass: keep `_adjacent_open_issues` under the 40-line function limit,
      no reuse of `_strip_markdown_code` (see proposal Risks)
- [ ] [AC1] Wire `spec-gate.yml`: `permissions: issues: read`, one `gh issue list`
      fetch into a file, pass `--adjacency-issues`

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] `shellcheck scripts/check-spec-gate.sh` passes
- [ ] `bats tests/*.bats` passes (no regression in the existing spec-gate suite)
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in with output produced this session
- [ ] PR opened referencing this spec folder, body carries `Refs #858` — **not**
      a closing keyword: #858 closes with the fixture-shape inventory in #857
