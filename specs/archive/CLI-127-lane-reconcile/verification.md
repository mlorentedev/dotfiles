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
- **The head PR's four-condition check (#2596, closed by the follow-up change).** `plan` now asks `prland.Blockers`, the read-only entry into `prland.Decide`, and prints each blocker under the head's command; `TestBlockers_*`, `TestCheckHead` and `TestLanePlan_PrintsTheHeadsLandingConditions` pin it. A draft is never the head (`TestStatus/ADraftIsNeverTheLandingHead`) and a detached worktree has no PR (`TestREST_ADetachedWorktreeHasNoPR`), both found by a read-only run on the live forge (lesson 455).
- **Session half covers every harness** through `RecentHarnessSession` (#2575, BUG-148): Claude Code, pi and Copilot.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-454-a-preview-and-the-verb-that-executes-it-must-ask-the-same-oracle.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-050 already carries the lane contract; this spec made no decision beyond it
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: the lessons concern one project's verbs; they recur in no second one yet

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-127-lane-reconcile/` -> `specs/archive/CLI-127-lane-reconcile/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)

### PR 3 live evidence (2026-10-16, read-only)

`dotf lane status --json` and `dotf lane plan --json` ran on this machine against the running herdr (none started): 20 lanes, 9 with a live session. The plan was 4 `adopt/ambiguous` and 4 `flag/ambiguous`; none of the 9 was `auto` or `pending`. The first run found two live worktrees (`feat/deploy-toml-paths`, `feat/worktree-harness-sessions`) classed `reap/pending`: the pending branch read time only, so the plan said "wait and it goes" for a worktree with a live session. Not `auto`, but not the AC4 wording either. Fixed in PR 2 (liveness is checked before any time-only reason, with a table case); the re-run is the evidence above. `apply` was not run live: a safety check is not performed by the reaper it guards against.

## Review round 1 (2026-10-16): FAIL, fixed

`review.md` (nan/qwen3.8-flash, reviewed `79cf7e5d`) found two Major and three Minor gaps. Fixed in one change:

- **F1 (Major), apply reaped under a different oracle than plan.** `lane.ForgeSweep` gives sweep the forge's merge answer; `TestApplyNeverReapsWhatPlanCalledAmbiguous` runs the real sweep over a real repository (lesson 454). Replacing the override with ancestry makes it fail (`changed=2`).
- **F2 (Major), the resolving command was unquoted.** `withPath` quotes with `humanPath`; a path no shell quotes alike gets no command and the reason says so. `TestPlanQuotesTheResolvingCommand`.
- **F3 (Minor), queue to lane matched by derived branch name only.** `Status` also skips a queued issue whose number a worktree already carries. `TestStatus/AHandNamedLaneIsNotAlsoPending`.
- **F6 (Minor), the four-condition deferral named no ticket.** It is #2596, above.
- **Handoff hygiene (Minor).** The promotion lines are answered above.

Not changed: the sequential `gh api` per branch (compliant with the proposal) and the `BranchPrefix` default `fix` (it only feeds the queued-branch derivation).

## Review round 1 follow-up (2026-10-16): #2596 and two live-run defects

`dotf lane plan` was run read-only against the live forge after the change. The first run, before the fixes, printed PR #2601 against three detached worktrees (an empty branch matched the newest PR) and named the draft #2582 the landing head. After the fixes the same run reads:

- the draft #2582 is listed but never the head, and the head's line is the first non-draft, #2600;
- under the head, `blocked:` names each unmet condition from `prland.Decide` (checks pending, merge state UNKNOWN), and the command follows;
- no detached worktree carries a PR.

`apply` was not run live (AC4 policy above). The independent review of the whole spec is re-run after this change lands.

## Review round 2 (2026-10-16): FAIL (nan/deepseek-v4-flash, reviewed `65d2486e`), fixed

Two Major and two Minor findings were real; F5 is a scope observation (the range aggregates unrelated commits by launcher construction), no action.

- **F1 (Major), the apply wiring was unguarded.** `realLaneVerbs` is the only place the sweep gets its forge and its session stores, and the oracle test built its own runner. `TestRealLaneVerbsSweepHonoursTheForgeAndTheSessions` drives the real wiring over a real repository and a pi session on disk. Mutations, each reverted: dropping `Forge: src.Forge` fails it (nil dereference), dropping `Sessions: sessionStores()` fails it (the live-session worktree is reaped). Lesson 457.
- **F2 (Major), the live-session resolver did not resolve.** `done` refuses a recent session whatever `--force` says. The row now carries `--force --ignore-recent-session` and says it strands the session; `TestPlanResolverForALiveWorktreeRunsThroughDone` runs the printed string through the real `worktree done` (mutation: without the flag it fails).
- **F3 (Minor), an adopt with nothing to run.** `planUnadopted` now falls back to `KindFlag` when the path cannot be quoted, as `abandon` does (`TestPlanResolversDoWhatTheyPromise`).
- **F4 (Minor), the dirty row hid its loss.** The reason now says the command discards the uncommitted changes (same test). This is the code route, so no contract file changed.

## Review round 3 (2026-10-16): PASS (nan/mimo-v2.6-flash, reviewed `7a49aa8e`)

No Blocker and no REAL Major. Four mutations by the reviewer each broke the test that guards the property (auto-only filter, liveness ordering, forge oracle and session stores in the real wiring). Dispositions:

- **AC4 wording (Minor).** "Every worktree with a live session is classed ambiguous" holds for the lanes plan acts on; a live `Running` lane gets no action by design and is never `auto`. The safety property, never reaped, is the tested one. Recorded here and not in the contract set, which stays untouched.
- **Branch not encoded in `REST.PRFor` (Minor, speculative).** Ticketed: https://github.com/mlorentedev/dotfiles/issues/2616.
- **Simultaneous `lane apply` (question).** Ticketed: https://github.com/mlorentedev/dotfiles/issues/2617, to be done after the atomic claim (#2551) since it changes the start half.
- **Scope of the range (Minor).** Launcher construction, same as round 2 F5. No action.
