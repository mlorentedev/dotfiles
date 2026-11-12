# run lanes-20261112t1126z

Armed 2026-11-12T11:26Z (the owner's go-ahead was 07:48Z; the arming turn waited 3h38m on the usage limit and the window kept the original expiry), expires 2026-11-12T19:48Z. Objective: work the lane/worktree follow-up queue in order (#3337, #3324, #3265+#3360, #3320), park the four that need a decision or a spec slot (#3114, #3307, #3327, #3318). Interim entries below; the disarm entry closes the file.

## Pre-arming: #3380 and the release

- #3380 (spec-gate warns instead of refusing a draft's unarchived spec) landed at `5dcfe6ff` after its triage; #3364 closed, worktree and branch removed, board Done. The owner cut v0.80.0; `dotf update` on this host reached 0.80.0 and its converge failed on Homebrew's `visual-studio-code` cask download, reported by dotf as `install dotf 0.80.0: install.sh: exit status 1`. Filed #3395 (the real failure is masked); no unattended retry of `system-upgrade`, which is outside the envelope.
- Peer envelopes read before arming: the v0.80.0 acceptance and the npm pins runs write `cli/internal/doctor`, the review run owns #3243; none touch the lane or worktree paths. `cli/internal/doctor/**` frozen here for that reason. #3391 was warned that main already holds lesson 590 (from #3380).

## Iteration 1: #3337 (lane doctor reports an unreadable gitdir)

- Branch `fix/3337-audit-unreadable-gitdir`, worktree `fix-3337-audit-unreadable-gitdir`. Red test first (`audit_test.go`: an admin dir whose `gitdir` file is gone is reported, not folded into a missing record; a second registration whose gitdir is gone is not listed as a blank duplicate), then the fix: `laneRecords` emits a finding with the `prune`/`repair` FIX and keeps the admin dir in the per-lane list, `located` drops empty-path entries before `auditLane` decides. 63 added lines over two files, under the threshold.
- PR #3397: PR-Agent on `550967ed` left one THEORETICAL (the empty-path placeholder padded the duplicate list; fixed by `located` with a test) and one REAL on the PR description (the body did not carry the red runs; reproduced with `dotf pr reproduce`, exit 1 on `550967ed`, control did-not-reproduce on `c74072b4`, body rewritten); triaged, landed with `dotf pr land` at `c74072b4` under the standing grant. #3337 closed by the merge; board Done; worktree and branch removed.

## Iteration 2: #3324 (rule 6b names the right checkout)

- Branch `fix/3324-detached-no-lease-message`, worktree `fix-3324-detached-no-lease-message`. Option B (message only, owner's answer): rule 6b tells a detached main checkout from a detached leaseless worktree, naming the worktree's path in the second case. Tests 6b (scratch detached worktree; asserts the message, the base repository and not the scratch path) and 6c (detached main checkout keeps its wording). The first assertion tripped on this run's own wording, `(main checkout: …)`, which was dropped for `($main_root)`.
- PR #3398: PR-Agent on `964f0839` left one THEORETICAL (an unasserted message segment) and a description-evidence row, both applied on `510656f5`; CodeRabbit posted a limit notice, not a review. Triage posted on `510656f5`; no reviewer ran on that head. Every check green there; scoped queue clear; landed with `dotf pr land` at `510656f5` under the standing grant. #3324 closed by the merge; board Done; worktree and branch removed.

## Iteration 3: #3265 and #3360 (Park under the 40-line limit)

- Branch `refactor/3265-park-path-under-40`, worktree `refactor-3265-park-path-under-40`, commit `5893c19c`: `Park` split into `parkPreflight`, `parkable`, `parkedLease`, `detachAtBase`, `releaseParkedLane`, `parkOutcome`; `parkSingleNamedLane` and `switchLaneLocked` (now `switchTarget`, `switchable`) use the same helpers. Two go/ast guards (`TestParkPathFunctionsStayUnderTheLimit`, `TestSwitchLaneLockedStaysUnderTheLimit`, limit 40) proved red by restoring `named.go` from HEAD (60 lines) and green on the new file. 234 production lines, under the cap; `go test ./internal/worktree ./internal/lane` ok, pinned lint 0 issues.
- Parked, branch local. The SDD gate counts the refactor over the 50-line threshold; the push under `SDD_LABELS=skip-sdd` was denied by the harness's own classifier (a gate bypass), and this run does not work around a denial. Two options on #3265 (the owner pushes under `skip-sdd`, there is no contract change; or a spec slot when one frees); board Blocked. #3360 builds on the same seam and waits on the answer.

## Iteration 4: #3320 (the prompt directory's own name)

- Branch `fix/3320-prompt-dir-name`, worktree `fix-3320-prompt-dir-name`. `PromptDirName = "dotf-lane-prompts"`, a one-time move of a legacy `dotf-lane` prompt directory (a regular file under the old name is left alone; both present, the new wins), `lane next` resolves through it; four tests. `git grep` outside the lane package finds no other reader of the old name (`.dotf-lane-prompt.md` is a different artifact).
- PR #3400: PR-Agent on `3b7c91ed` found one REAL, a dry `lane next` moved the directory. Reproduced with `dotf pr reproduce` (exit 1 on `cd91fa52`, the red test `TestLaneNext_DryRunLeavesTheLegacyPromptDirectoryAlone`), fixed on `ce332d91`: `PromptDir(commonDir, migrate)` with `lane next` passing `!dryRun`. The THEORETICAL race (two movers, the loser's rename fails) applied in the same commit: a rename failing with ENOENT while the new directory exists resolves to it. Triage posted on `3b7c91ed` and disclosed that the fix head was unreviewed; every check green on `ce332d91`, no reviewer ran on it in the window, scoped queue clear; landed with `dotf pr land` at `ce332d91` under the standing grant. #3320 closed by the merge; board Done; worktree and branch removed.

## Parked without an attempt (one ERROR/WHY/FIX comment each, board Blocked)

- #3114: 69 executable lines over the threshold, no spec slot (the free one is #3227's), `skip-sdd` is the owner's call.
- #3307: the lease's home (next to the identity record, ADR-061 territory, or re-created on switch) is a design choice.
- #3327: `dotf doctor` sweep entry; the two options measure different things and `cli/internal/doctor` is written by two peers tonight.
- #3318: needs the ADR extending ADR-061 the issue itself asks for.

## Disarm

**run lanes-20261112t1126z — ended queue-exhausted after 4 iterations / 11:26Z to 12:09Z (go-ahead at 07:48Z; 3h38m lost to the usage limit before arming) — 3 done, 1 parked with work, 4 parked without an attempt, 3 merged, 0 unmerged.** Merged on main: #3397 (`c74072b4`), #3398 (`510656f5`), #3400 (`ce332d91`). Waiting on a human: the #3265 A/B (its branch `refactor/3265-park-path-under-40` at `5893c19c` exists only in the local worktree `refactor-3265-park-path-under-40`, push denied), and the four A/B parks above; #3360 waits on #3265. Filed: #3395. Host: the wake-lock held for the window. Nothing irreversible is pending; the three merges are the irreversible part and all are on main.

## Post-disarm, with the owner back

- Owner's answers: #3265 and #3114 under `skip-sdd` (no contract change), #3307 direction 1 (lease next to the identity record, ADR + spec) and #3327 option A (the whole seat, spec), both waiting on a spec slot; the one free slot went to the review run's BUG-170 (#3243) with the owner's confirmation. Decisions recorded on each issue.
- #3407 (#3265): pushed under the owner's `skip-sdd`; PR-Agent found two REAL (the guards passed exactly 40 lines against AGENTS.md's under-40; reproduced by grep on `c06f5a17`, fixed to `>=`) and one THEORETICAL (branch deletion moved between the prompt removal and the claim release in the sweep's park; now last, the order `Park` has) plus a harness-compliance REAL on the body (per-file breakdown added, reproduced the same way). Landed at `434dd963` with the fix head unreviewed (disclosed); #3265 closed, worktree removed.
- #3408 (#3114): reusing `LinkedWorktrees` brought the issue's 69-line prototype to 37, under the threshold, no label needed. PR-Agent's THEORETICAL that the `done main` test could not fail was right: it now adds a linked worktree and asserts the not-found error, proven red with the lookup widened (`e5ef10c0`). Landed at `e5ef10c0`; #3114 closed, worktree removed.
- #3416 (#3360): option A, `sweepOne`/`sweepLane`/`SweepReport.record`, 121 gate lines under the owner's `skip-sdd` (asked and answered in session). PR-Agent: no key issues. Landed at `c70d6089`; #3360 closed, worktree removed.
- Still parked: #3307, #3327 (spec slot), #3318 (ADR). Filed this run: #3395.

## Correction: head SHAs versus the squash commits on main

The entries above name each PR's head commit. `dotf pr land` squashes, so main carries a different SHA for each: #3397 head `c74072b4`, on main as `eb4d3788`; #3398 head `510656f5`, on main as `79791e38`; #3400 head `ce332d91`, on main as `f70dfb02`; #3407 head `434dd963`, on main as `f28c4370`; #3408 head `e5ef10c0`, on main as `965645b5`; #3416 head `c70d6089`, on main as `85f63e37`. Main after the last landing: `85f63e37`.
