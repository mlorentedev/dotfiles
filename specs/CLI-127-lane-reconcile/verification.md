---
tags: [spec, verification, templates]
created: "2026-10-15"
---

# Verification - CLI-127-lane-reconcile

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (PR 1, `lane status`) -> `TestDerive` (one row per phase, herdr absent in several) and `TestStatus` (subtests: the 9 phases through the join, herdr absent, unknown liveness, landing head); the f1 verification line passes. `TestLaneStatus*` in `cli/internal/cmd` pins the `--json` and table shapes.
- [x] AC2 (PR 2, `lane plan`) -> `TestPlan` (one row per D1 case, the D2 rows, the D3 landing row) and `TestPlanMutatesNothing` (a real git fixture, a non-empty plan, a hash of refs, worktrees, statuses and leases before and after); the f2 verification line passes. `TestLanePlan*` in `cli/internal/cmd` pins the text and `--json` shapes.
- [x] AC3 (PR 3, `lane apply`) -> `TestApply` (auto actions only, one sweep and one next, a second apply over the post-state is `changed=0`, a printed-only start is not a change, the first verb error stops the run) and `TestLaneApply_PrintsChangedAndRunsOnlyAutoVerbs`; the f3 verification line passes.
- [x] AC4 (PR 3, live process never reaped) -> `TestApplyNeverReapsALiveWorktree` (liveness `live` and `unknown` on a merged, clean, expired worktree: ambiguous, and apply never reaches sweep); the f4 verification line passes. Live half: see Test status.

## Test status

- PR 1: `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` all pass; `golangci-lint run` (2.12.2, the pin) reports 0 issues.
- PR 2: the same suite passes. `dotf lane plan` ran live and read-only on 2026-10-16: no worktree with a live session was classed `auto` (the merged ones with a running lease are `pending`, the unleased ones `ambiguous`).
- Manual smoke test (read-only, against the running herdr, no server started): `dotf lane status` on this machine listed 12 lanes in about 10s wall time (network-bound: one REST call and one `herdr worktree list` per branch). The first run found the landing-head defect now pinned by `TestStatus/LandingHeadIsPickedAmongReviewLanes` (lesson 450).
- No regressions in existing test suite: yes.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Landing order** is the lowest-numbered open PR among the lanes that derived to Review: FIFO by PR number, an interim until ADR-049's queue defines it. An unadopted worktree's PR is not in it.
- **No lease outranks everything.** A worktree with an open or merged PR but no `.dotf-worktree.json` reads Unadopted, not Review or Landed, until `worktree adopt` (D2: adoption is never `auto`).
- **Merged comes from the REST forge alone** (`lane.ForgeGit`). Ancestry is dropped because a branch with no commit yet reads as merged; the cost is that a branch merged without a PR reads unmerged, which is the safe side because `sweep` re-checks its own gates.
- **Liveness has three answers.** `Combine` of a process scan (`worktree.ProcessInside`, a new read-only export in `process_export.go`, agreed with CLI-118 on #2550) and a session transcript. A probe that cannot answer (no scan on Windows, unreadable store) is `unknown`, shown as such, and reads as live for the phase. It never feeds an auto action.
- **`plan` mirrors sweep through `Row.State`**, the lifecycle state `Classify` already computed: REAPABLE with an idle liveness is `auto`; ACTIVE for a time-only reason (lease running, under 15 minutes old) is `pending` with the later of the two as its due time; liveness `live` or `unknown`, DIRTY, a reap hold, Stalled and Orphan are `ambiguous` with `dotf worktree done --force <path>`.
- **A start is `auto` only while no lane is in flight**, and only for the first pending item in queue order, so a second apply finds the lane the first one started (lesson 451).
- **The head PR's four-condition check is not printed yet.** `plan` prints the landing order and the head's command (`dotf pr triage-queue --pr N && dotf pr land N`, D3). Evaluating the four conditions needs an exported read-only fact reader in `cli/internal/prland`, which is outside this run's write paths; it is ticketed rather than copied, so the landing conditions keep one implementation (`prland.Decide`).
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

### PR 3 live evidence (2026-10-16, read-only)

`dotf lane status --json` and `dotf lane plan --json` ran on this machine against the running herdr (none started): 20 lanes, 9 with a live session. The plan was 4 `adopt/ambiguous` and 4 `flag/ambiguous`; none of the 9 was `auto` or `pending`. The first run found two live worktrees (`feat/deploy-toml-paths`, `feat/worktree-harness-sessions`) classed `reap/pending`: the pending branch read time only, so the plan said "wait and it goes" for a worktree with a live session. Not `auto`, but not the AC4 wording either. Fixed in PR 2 (liveness is checked before any time-only reason, with a table case); the re-run is the evidence above. `apply` was not run live: a safety check is not performed by the reaper it guards against.
