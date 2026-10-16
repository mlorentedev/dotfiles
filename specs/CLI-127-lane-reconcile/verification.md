---
tags: [spec, verification, templates]
created: "2026-10-15"
---

# Verification - CLI-127-lane-reconcile

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (PR 1, `lane status`) -> `TestDerive` (one row per phase, herdr absent in several) and `TestStatus` (subtests: the 9 phases through the join, herdr absent, unknown liveness, landing head); the f1 verification line passes. `TestLaneStatus*` in `cli/internal/cmd` pins the `--json` and table shapes.
- [ ] AC2 (PR 2, `lane plan`) -> not started
- [ ] AC3 (PR 3, `lane apply`) -> not started
- [ ] AC4 (PR 3, live process never reaped) -> not started. The read-only half ran on 2026-10-16, see Test status.

## Test status

- PR 1: `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` all pass; `golangci-lint run` (2.12.2, the pin) reports 0 issues.
- Manual smoke test (read-only, against the running herdr, no server started): `dotf lane status` on this machine listed 12 lanes in about 10s wall time (network-bound: one REST call and one `herdr worktree list` per branch). The first run found the landing-head defect now pinned by `TestStatus/LandingHeadIsPickedAmongReviewLanes` (lesson 449).
- No regressions in existing test suite: yes.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Landing order** is the lowest-numbered open PR among the lanes that derived to Review: FIFO by PR number, an interim until ADR-049's queue defines it. An unadopted worktree's PR is not in it.
- **No lease outranks everything.** A worktree with an open or merged PR but no `.dotf-worktree.json` reads Unadopted, not Review or Landed, until `worktree adopt` (D2: adoption is never `auto`).
- **Merged comes from the REST forge alone** (`lane.ForgeGit`). Ancestry is dropped because a branch with no commit yet reads as merged; the cost is that a branch merged without a PR reads unmerged, which is the safe side because `sweep` re-checks its own gates.
- **Liveness has three answers.** `Combine` of a process scan (`worktree.ProcessInside`, a new read-only export in `process_export.go`, agreed with CLI-118 on #2550) and a session transcript. A probe that cannot answer (no scan on Windows, unreadable store) is `unknown`, shown as such, and reads as live for the phase. It never feeds an auto action.
- **Session half covers every harness** through `RecentHarnessSession` (#2575, BUG-148): Claude Code, pi and Copilot.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-127-lane-reconcile/` -> `specs/archive/CLI-127-lane-reconcile/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
