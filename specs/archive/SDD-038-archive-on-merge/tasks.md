---
tags: [spec, tasks, templates]
created: "2026-08-06"
---

# Tasks - SDD-038-archive-on-merge

> TDD order. One task = one focused commit.
>
> `[P]` = no dependency on another unchecked task. `[AC<n>]` = satisfies acceptance criterion #n in `proposal.md`.

## Setup

- [x] Branch created from main: `feat-spec-gate-archive-on-merge`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] Work-gate verified: issue #670 is OPEN (`dotf spec init --issue 670`)

## Implementation

- [x] [AC1] Failing test: PR closes an issue whose active spec is not archived -> exit 1, message names `dotf spec archive`
- [x] [AC1] Parse closing keywords from `SDD_PR_BODY`; build the issue -> active-spec map from the base ref; assert `specs/archive/<id>/` at head
- [x] [P] [AC2] Failing test: the same PR passes once the folder is archived
- [x] [P] [AC3] Failing test: `Refs #N` / `Part of #N` / a bare `#N` in prose do NOT trigger the check
- [x] [P] [AC4] Failing test: all three `issue:` frontmatter shapes match (`owner/repo#N`, `repo#N`, bare `N`), plus the full-URL closing form
- [x] [P] [AC5] Failing test: a cross-repo closing reference (`Closes owner/other#5`) is ignored
- [x] [P] [AC6] Failing test: the check fires below the LOC threshold, and is not skipped by `skip-sdd`
- [x] [P] [AC7] Failing test: `skip-archive` + rationale passes; the label alone fails
- [x] [P] [AC8] Failing test: empty PR body, and a closed issue with no matching active spec, are clean passes
- [x] [AC9] Confirm the pre-existing `tests/check-spec-gate.bats` suite is still green

## Closing

- [x] Every acceptance criterion covered by at least one test
- [x] Every acceptance criterion has a `features.json` entry with a non-vacuous verification command
- [x] Lint passes (`shellcheck -x`)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Follow-up (out of scope here, tracked separately)

- [ ] One-time sweep: archive the ~21 specs whose issue is already closed (#670 remains open until this lands; the sweep PR is what closes it, and must archive THIS spec to pass the new gate)
- [ ] Backfill `issue:` frontmatter into the 28 active specs that lack it
- [ ] Rename stragglers (`specs/CLI-019`, no slug); resolve the `AI-022-*` id collision and the duplicate `MEMORY-001-*` pair

## Machine-readable features

See sibling `features.json`.
