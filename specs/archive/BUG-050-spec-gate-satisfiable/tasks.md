---
tags: [spec, tasks, templates]
created: "2026-08-07"
---

# Tasks - BUG-050-spec-gate-satisfiable

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Worktree created from `origin/main`: `dotfiles-wt-800`, branch `fix/spec-gate-satisfiable`
- [x] Gating issue open: dotfiles#800
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

Measurement came before design here. The issue's own proposed fix ("count
`specs/archive/<id>/` as a touch") was checked against the real mechanism and
found insufficient — the gate accumulates LOC against a floor, and an archive
move does not clear it. That measurement is what AC1 rests on.

- [x] Measure how an archive move actually renders: `git show --numstat 9b24cce`
      (PR #787) → 3 files at `0 0`, `proposal.md` at `3 1` = **4 LOC** vs
      `SPEC_FLOOR=10`. Confirms the touch must be set outright, not via `SPEC_LOC`.
- [x] [AC1] Write failing test: a large PR archiving the spec of the issue it
      closes passes the gate.
- [x] [AC3] Write failing test: a spec created and archived in the same PR counts.
- [x] [AC2] Write test (green before AND after — a preserved protection, not a
      new behaviour): a gratuitous archive-move earns no touch.
- [x] [AC4] Write test (green before and after): no closing keyword, no mandate.
- [x] [AC1] `_archived_spec_issue_map()` — mirror of `_active_spec_issue_map()`
      over `specs/archive/<id>/proposal.md`, same `if`-not-`&&` loop-body
      discipline so a spec without an `issue:` field cannot fail the whole loop.
- [x] [AC1][AC2] `_mandated_archive_ids()` — intersect the archived map at head
      with the PR's closing issue numbers.
- [x] [AC1] `_is_mandated_archive_path()` + the `elif` branch in the diff walk;
      sets `SPEC_TOUCHED=1` directly rather than adding to `SPEC_LOC`.
- [x] [AC1] `--explain` lists the mandated archives, so a pass is inspectable
      rather than mysterious.
- [x] [AC5] Verify red: 3 of the 5 new cases fail against the unfixed gate; the
      2 protection-preserving cases pass both before and after.

## Closing

- [x] Every acceptance criterion is covered by at least one test
- [x] Lint passes (`shellcheck scripts/check-spec-gate.sh`)
- [x] Syntax check passes (`bash -n`)
- [x] Full suite: no new failures (2 pre-existing, tracked as #755 and #807)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder
- [x] Spec archived in the same PR — this change's own subject matter
