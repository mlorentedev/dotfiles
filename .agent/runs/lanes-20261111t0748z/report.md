# run lanes-20261111t0748z

Armed 2026-11-11T07:48Z, expires 2026-11-11T19:48Z. Objective: land HARNESS-187 (#3261), then #3262, #3263, #3265, #3293 in order. Interim entries below; the disarm entry closes the file.

## Iteration 1: #3261 (HARNESS-187, lane identity in the worktree git dir)

- Branch `feat/3261-lane-identity`, worktree `feat-3261-lane-identity`. Nine commits through `5dca2592`+1; production lines 172 of the 300 cap, 13 files.
- Second advisory pass found two defects in the first implementation, both fixed before review: a refused `lane init` wrote the marker first (guard armed with no lanes), and `lane switch`/`lane park` still reached another repository's lane by path. The AC3 test asserts both.
- Full Go suite ok, pinned lint 0 issues on both OSes, shellcheck clean.
- Follow-up filed: #3318 (two repositories sharing a basename cannot both have lanes).
- Pooled review round 1 (nan/glm5.3-flash): FAIL on one REAL Major, `lane.Identity` erroring on a main checkout whose common git dir holds the `dotf lane next` prompt directory of the same name; fixed with a regression test. Round 2 (nan/mimo-v2.6-flash): PASS; two minors applied (unreadable record fails closed; both sides read one line), one ticketed, rest declined with reasons in verification.md.
- Follow-ups filed: #3320 (dotf-lane name shared by the prompt directory and the record; on the board as HARNESS-189, P3), #3324 (rule 6b message names the main checkout for any detached leaseless worktree; board fields pending, GraphQL quota exhausted when filed).
- Spec archived; PR #3325 opened as a draft and marked ready once the shared GraphQL quota allowed it; CI and reviewer triage follow.
- Peer coordination: feat-worktree-one-root session took lesson 580 on #3312; this run holds 579 and ADR-061.

## Iteration 2: #3262 (HARNESS-190, dotf lane doctor)

- Branch `feat/3262-lane-doctor`, worktree `feat-3262-lane-doctor`, stacked on `feat/3261-lane-identity` until #3325 lands; rebased onto main after that, then reviewed and archived, since the review binds to a commit.
- The committed envelope does not list `cli/internal/doctor/**`, so the check lives in `cli/internal/lane` (`AuditLanes`) behind `dotf lane doctor`, the home the issue allows; the `dotf doctor` sweep entry is #3327 with two options. Production lines 254 of the 300 cap, 11 files.
- Measured on a scratch repository before writing: after `mv` of a lane, git keeps the admin dir with a `gitdir` pointing at the old path and the record intact, lists the old path as prunable, and refuses `worktree add` at it. The audit enumerates `<repo>/.git/worktrees/*/` for that reason (lesson 582; main took 581 during the review), and `gitdir` is relative under `worktree.useRelativePaths` (git 2.54 measured).
- Advisory pass before review found four gaps, all applied: a second reader of the record (extracted `readIdentityRecord`, shared with `Identity`, unreadable is a finding), the relative gitdir, an empty `WORKTREE_ROOT` resolving to the cwd, and drift tests asserting full paths that only pass on macOS by substring luck.
- Spec HARNESS-190 at `verifying`; features f1..f4 verified; pinned lint 0 issues on both OSes; full suite pending at the time of this entry.
- Full Go suite exit 0 (55 packages ok on `047ccc6b`, `$?` recorded without a pipe). Pooled review launched on the stacked branch (scope `658ef64b...18d71323`, reviewer agy/gemini-3.8-flash-high, random draw) while #3325 waited on the GraphQL quota: the rebase onto main after #3325 lands is a replay of the same seven commits, so the reviewed diff is the one that ships; the archive gate's digest check decides whether the review binds.
- #3325 first `from-zero (ubuntu-latest)` attempt went red on "the second converge was not a no-op: tools upgraded pi". Root cause measured: pi v1.5.2 was published 2026-11-10T12:25:43Z and the two converges ran at 12:25:00Z and 12:26:45Z the next day, either side of mise's 24h `minimum_release_age`; the second converge's upgrade was correct. Attempt 2 green on the same head. Filed #3331 (two options: filter that change in the gate, or run the second converge with upgrades off). Every other check green on `658ef64b`; no reviewer output yet while the PR stays a draft.
- #3325 review: PR-Agent round 1 (three inline findings on `658ef64b`) and CodeRabbit (eight on `a793ffd4`) all dispositioned; the REAL one (a dangling symlink at a lane path slipped past the preflight and the marker was still written) fixed in `a793ffd4` + `e7921282` (Lstat, marker written last, uninspectable paths are errors). PR-Agent round 2 on `e7921282` added a REAL on symlinked-lane adoption, refuted by `8be26100`'s characterisation test (did-not-reproduce), a THEORETICAL on the `dotf-lane` directory name (skip, #3320), and a harness-compliance row answered by the triage record itself. Triage posted at 13:5xZ on head `8be26100`; landing waits on `integration`, `from-zero` and the macOS/Windows test legs.
- Reproduction evidence corrected before posting: the first `reproduced` line (on `3495bea8`) was invalid, since `go run -C cli ./cmd/dotf pr reproduce` runs the check in `cli/` and `go test -C cli` from there fails on `chdir`, exit 1, which `reproduce` reads as the defect. Re-run from `cli/` without `-C`: the probe restoring `named.go` to `658ef64b` reproduces both marker tests (exit 1), the adoption test does not reproduce (exit 0). Filed #3333 (two options: a control run on the clean head under `--probe`, or output signatures of a check that never started); on the board as bug P3.
- `feat/3262-lane-doctor` rebased onto `8be26100` (ten commits, lane and cmd packages ok); the rebase onto main, review round 2 and archive follow the landing of #3325.
- #3325 landed at `8be2610` (standing grant: CI green on the head, clean, triage posted, scoped queue clear); `feat-3261-lane-identity` worktree and branch removed; #3261 closed by the merge.
- HARNESS-190 review round 2 (nan/mimo-v2.6-flash, on `cb655512` after the rebase onto main): FAIL on one REAL Major, a fresh clone whose lane paths hold another repository's lanes got the plain-directory FIX ("move aside, prune"), which dismantles the other repository's lanes. Advisor: fix the class (a FIX attached to every refusal of `assertWorktreeOf`), measure the stale-registration state before writing the strings. Measured on a scratch pair: `git worktree prune` keeps the stale entry while the other `.git` exists, `remove --force` refuses, and `git worktree repair` from the wrong repository rewrites the other worktree's `.git` file (hijack). Fix `9fbb5452` behind red `3ccf8228`: the move and prune only for a plain directory (`errNotAWorktree`), a foreign worktree is left to its owner with #3318 named, a stale admin dir is named for a hand delete; the test runs the named repair and asserts both repositories audit clean. Three Minors applied (three assertion-only tests, `runLaneDoctor` extracted, evidence re-anchored on test names). Budget: 297 of 300 after dropping a redundant `Getwd` and shortening `auditRoots`, `undeclaredRecords` and `sameSet`; 14 of 14 files. Lesson renumbered to 582 (main took 581 while the branch was in review). Round 3 launched on `b6d74f79`.
- HARNESS-190 review round 3 (nan/glm5.3-flash, on `b6d74f79`): PASS; the reviewer mutation-tested the round-2 fix. Two Minors: an admin dir with an unreadable `gitdir` is skipped rather than reported (ticketed #3337, chore P3: four production lines against a 297-of-300 budget), and the exit-2 path missing from AC4 (runbook 7.6 names its causes; the contract set is closed). Spec archived to `specs/archive/HARNESS-190-lane-doctor/` with the three review rounds; branch pushed; PR #3338 opened as a draft, CI on `af40a0a9` pending at the time of this entry.
- #3338 `integration` red twice on `af40a0a9`: the Docker build cannot resolve `archive.ubuntu.com` (`apt-get update failed 3 times`), and main's own latest run (`4e0281f6`) failed `test` and `integration` the same way (`Temporary failure resolving 'azure.archive.ubuntu.com'`, "Install zsh" timed out), so it is the runner's network, not the change. One more `rerun --failed` queued once the run completes.

## Iteration 3: #3263 (sweep says why it did not park a lane)

- Branch `fix/3263-sweep-skip-reason`, worktree `fix-3263-sweep-skip-reason`, from `4e0281f6`. Scope limited to what #3310 left open: the reason. Two commits (`9a52fd06` test, `6c186810` fix): `parkSingleNamedLane` returns `(parked, why)`, `laneLanded` carries the landing decision on its own (also the seam #3265 asks for), `SweepReport.LaneSkips` (`lane_skips`, omitempty) lists each lane with its reason, the CLI prints them with the session and merge-unknown skips. Reasons: dirty tree, `branchLanding`'s why (fallback "its landing could not be decided"), lease write error, and git's first stderr line on a refused detach (`CombinedOutput`; `.Run()` only had the exit status). Sweep-level test on the real fixture: the lane's branch merged and pushed so the ancestry fast path classifies it parkable, an `index.lock` makes the detach fail; path compared with `samePath` for the Windows leg. `go test ./internal/worktree/ ./internal/cmd/` ok, vet and pinned lint 0 issues on both OSes; 33 added executable lines.
- Parked. The pre-push SDD gate counts 66 LOC and refuses the push without a spec; `dotf spec init HARNESS-191-sweep-skip-reason --issue 3263` refuses at the WIP limit (10 active specs on main). The change adds a JSON key, a public contract, and the standing rule says an unattended run does not land that under `skip-sdd`. Two options on #3263 (free a spec slot or `--over-wip-limit`, then review and archive in the PR; or push under `skip-sdd` with the rationale); board Status set to Blocked. The branch stays local in its worktree until the answer.
- #3293: scope-split comment posted. The squash-merge half is #3310 and #3263; the identity half is parked with two options (a provenance record per worktree, the ADR-061 shape; or namespaced app roots enforced by doctor). Board untouched (never claimed).
- #3265 not started: it builds on `laneLanded` from the parked branch, so it waits on the #3263 answer. Queue is exhausted for this run; what remains is landing #3338.
- #3338 attempts 3 and 4 failed the same way (`test`, "Install zsh": `Temporary failure resolving 'azure.archive.ubuntu.com'`; `integration` went green on attempt 3). Every branch's run since 13:53Z fails there, so the breaker applies: no more immediate reruns. Filed #3339 (chore P3, two options: fall back to the canonical mirror when the Azure one does not resolve, or stop installing zsh on the runner); `.github/**` is frozen for this run, so the fix is not attempted here. One deferred rerun of the failed job queued for ~15:30Z; #3338 stays a draft until its head is green.
- #3338 attempt 6 (17:02Z) green on `af40a0a9`; marked ready at 17:04Z. PR-Agent full review at 17:10Z: no findings, three ticket-compliance rows (git older than 2.48 skip, #3327 and #3337 defer); CodeRabbit hit its free-tier limit at 17:04Z and published no review. Triage posted, scoped queue clear, landed with `dotf pr land` at `9431209a` (`af40a0a` squash) under the standing grant; #3262 closed by the merge. `feat-3262-lane-doctor` worktree and branch removed.

## Disarm

**run lanes-20261111t0748z — ended queue-exhausted after 3 iterations / 9h37m (07:41Z to 17:18Z) — 2 done, 2 parked, 2 merged, 1 unmerged.** Waiting on a human: the #3263 A/B and the #3293 A/B. Nothing irreversible is pending; the two merges (#3325, #3338) are the irreversible part and both are on main.

### Waiting on you, irreversible first

1. **Landed on main:** #3325 (`8be2610`, HARNESS-187, lane identity in the worktree git dir, ADR-061) and #3338 (`9431209a`, HARNESS-190, `dotf lane doctor`). Both under the standing grant: CI green on the head SHA, clean, `## Review triage` posted, scoped queue clear. #3338 landed reviewed by PR-Agent and three pooled adversarial rounds; CodeRabbit published no review (free-tier limit), disclosed in the triage.
2. **#3263, parked (Blocked):** fix written and green on `fix/3263-sweep-skip-reason` (`9a52fd06` test, `6c186810` fix, `c39d7c7c` lesson 584), local only, in `/Users/manu/Projects/worktrees/dotfiles/fix-3263-sweep-skip-reason`. Do not `worktree done` it. Question: (A) free a spec slot or `--over-wip-limit`, then `dotf spec init HARNESS-191-sweep-skip-reason --issue 3263`, review and archive in the PR; or (B) push under `skip-sdd` with the rationale (33 added executable lines, additive `omitempty` JSON key). Cheapest unblock: answer on the issue; the branch pushes the moment either is chosen.
3. **#3293, parked by scope (Backlog, never claimed):** the squash-merge half is #3310 + #3263; the identity half asks (A) a provenance record per worktree (the ADR-061 shape) or (B) namespaced app roots enforced by `doctor`. Nothing is built for it.

### Parked, with the question restated

- #3263: see item 2 above. Rung reached: implementation complete, push refused by the SDD gate (66 gate lines against 50), `spec init` refused at the WIP limit (10 of 10). The run's rule says an unattended session does not land a public-contract change under `skip-sdd`.
- #3293: see item 3. Rung reached: scope split only.
- #3265 (extract `Park`/`parkSingleNamedLane`/`switchLaneLocked` under 40 lines): not started; `laneLanded` on the parked branch is its first seam, so it waits on the #3263 answer. Not claimed, Backlog.

### Done but unmerged

- `fix/3263-sweep-skip-reason`, three commits, tests re-run green on the final tree (`go test ./internal/worktree/ ./internal/cmd/` ok, `GOOS=windows go vet` ok, pinned lint 0 issues on both OSes).

### Evidence index

| Item | Command | Exit | Where |
|---|---|---|---|
| #3325 | `cd cli && go test ./... -count=1`; `golangci-lint run` both GOOS | 0 | PR #3325 body, `specs/archive/HARNESS-187-*/verification.md` |
| #3325 land | `dotf pr triage-queue --pr 3325 && dotf pr land 3325` | 0 | PR #3325 `## Review triage` comment, merge at `8be2610` |
| #3338 | `go test ./internal/lane/ ./internal/cmd/`; lint both GOOS | 0 | PR #3338 body, `specs/archive/HARNESS-190-lane-doctor/verification.md` |
| #3338 CI | run 43039304975, attempts 1 to 6 (`test` and `integration` red on the Azure apt mirror until attempt 6) | green on attempt 6 | #3339 names the logs |
| #3338 land | `go run ./cmd/dotf pr triage-queue --pr 3338` then `pr land 3338` | 0, `[MERGED] #3338 at af40a0a` | PR #3338 `## Review triage` at 17:1xZ |
| #3263 | `go test ./internal/worktree/ ./internal/cmd/ -count=1`; `GOOS=windows go vet`; pinned lint | 0 | worktree `fix-3263-sweep-skip-reason`, branch at `c39d7c7c` |
| #3263 push | `git push -u origin fix/3263-sweep-skip-reason` | 1 (SDD spec-gate, 66 LOC) | issue #3263 park comment |
| #3263 spec | `go run ./cmd/dotf spec init HARNESS-191-sweep-skip-reason --issue 3263` | 1 (10 active specs, limit 10) | same |
| Repo-wide queue at disarm | `go run ./cmd/dotf pr triage-queue` | 1: only #3334, not this run's PR | this file |

### Budget spent

3 iterations, 9h37m wall clock of a 12h window, 5 queue items (2 landed, 1 parked done, 1 parked by scope, 1 not started), 8 tickets filed (#3318, #3320, #3324, #3327, #3331, #3333, #3337, #3339), all on the board with Status, Priority and Type. No dollar figure exists for this run; the pooled reviews drew on the NaN pool (`ai/nan/README.md` has the rates) and none is attributable here.

### Host-suspend line

The host stayed awake: `caffeinate -i -s -t 43200` armed at 07:41Z and still running at disarm. The window was not fully covered by the run itself: from roughly 15:25Z to 17:01Z the session was paused by the harness's usage limit, so #3338's sixth attempt ran at 17:02Z instead of ~15:50Z. No work was lost; one rerun was delayed.

### Knowledge

- Lesson 582 (landed with #3338), lesson 584 (on the parked branch), ADR-061 (landed with #3325).
- Vault: the autopilot skill's third eligibility veto now judges the WIP limit at selection, by the gate's own count (`00_meta/skills/autopilot/SKILL.md`, vault `8fc6d656`); reaches the repo copy on the next `dotf harness refresh` (`harness/**` is frozen for this run).

### The one next action

Answer #3263: A or B. Everything else in this line of work (#3265, the #3293 identity half) follows from that branch landing.

### Post-disarm: main after #3338

- `ci.yml` on main at `9431209a` (run 43065883128): `test` and `integration` failed on the Azure apt mirror again (`Temporary failure resolving 'azure.archive.ubuntu.com'` in "Install zsh"; the container build's `apt-get update` likewise), the #3339 signature, not the change: the same head passed both jobs on PR #3338's attempt 6 at 17:02Z. One `rerun --failed` issued; the evidence is on #3339.
- Rerun (attempt 2) of run 43065883128 green: main at `9431209a` passes `test` and `integration`. Nothing landed by this run is red on main.

### Post-disarm: #3263 reopened on the spec route (2026-11-12)

The owner chose A for #3263 (spec, never `skip-sdd`) and A for #3293 (a provenance record per worktree, the ADR-061 shape; recorded on the issue, nothing built, waits for a WIP slot).

- `HARNESS-191-sweep-skip-reason` opened over the WIP limit (`--over-wip-limit`, reason in its frontmatter), filled and verified; PR #3355 (draft) from `fix/3263-sweep-skip-reason`; #3263 In Progress on the board.
- Review round 1 (`e02600e0`): FAIL, two REAL Majors. F1: the command test compared an OS-separator path with git's slash path, red on `test (windows-latest)` only. F2: lesson 584 was already taken on `main` (landed by another PR between the lesson's commit and this push), `hygiene` and `knowledge-gate` red. Six Minors. All applied on `4ced9f9f` except the `SweepWithRunner` length, ticketed as #3360 (on the board; option A folds it into #3265). Dispositions in `verification.md`.
- A nil landing reader is unreachable in production: the real runner reaches it in `list.go` and panics before `laneLanded`; the guard added there is tested through the mock runner.
- Review round 2 (`94cb2f6f`): FAIL on one REAL Major that was true: the Windows proof for round-1 F1 did not exist, because the pre-push archive-on-merge gate refused every push while the PR closed #3263 with the spec active. Three artifacts each correct and no path through: ticketed as #3364 (two options; A moves the hard refusal to CI's `spec-gate`, where the invariant belongs). The way through used here: the PR body switched to `Refs #3263` (the gate's own non-closing path) for the push of `564f4a2c`, to be restored to `Fixes #3263` with the archive. Round 2's three Minors applied (`laneLanded` keeps the re-check it had; an unreadable status says so instead of claiming a dirty tree) or declined with the reason in `verification.md`.
- The rebase onto `main` was not done: the force-push it needs was denied in this session; the branch is 5 commits behind `main` with no conflicts, and `dotf pr land` decides whether that base moved too far.
- Both pre-existing queue entries (#3354, #3353) belong to other threads on this host with their own worktrees; not touched here.
- Review round 3 (`14541186`): PASS, two Minors (function length, ticketed; an unticked `tasks.md` box, declined so the digest stays). Spec archived in `c70410ec`; the PR body went back to `Fixes #3263`; `gh pr ready`.
- PR review: PR-Agent (four inline, one REAL reproduced with a probe and pinned by a new test, one REAL refuted on the rule's own scope, one deferred to #3265/#3360) and CodeRabbit (three inline, all applied: a dirty named lane never reached the park path because `Classify` stops it at gate f, so the loop now records it with `Classify`'s reason; two lesson-585 corrections). Triage posted; `dotf pr triage-queue --pr 3355` exit 0.
- **#3355 landed** with `dotf pr land` at `885839c9` -> main `ec724137`; #3263 closed, Done on the board; worktree and branch removed.
- The repo-wide queue still lists #3354 and #3353, other threads' PRs on this host.

### The one next action (updated)

#3265: extract the park family under 40 lines from the `laneLanded` seam, with characterisation tests first; #3360 folds into it. #3293 waits for a WIP slot.

### Post-land: a defect the last push introduced (#3372, #3374)

A second opinion on the landed change found that `885839c9`'s last commit, applied after the final review round, recorded every dirty named lane in `lane_skips`, including a lane whose lease was live or under fifteen minutes old. A sweep then printed `skipped: <lane>: has uncommitted changes or untracked files` for a lane someone was working in, which reads as a complaint about live work. Filed as #3372 (bug, P3).

- Fix in #3374 from `fix/3372-dirty-live-lane` (`98a22323`): `laneHeldByDirt` records a dirty lane only when it is not parked, its lease has expired and it is older than fifteen minutes, the same three conditions `Classify` applies before deciding a lane is parkable. New test `TestSweepDoesNotReportADirtyLaneWithALiveLease`; the existing expired-lease test keeps the entry. Under 20 lines, obvious cause: no spec.
- CI 27/27 green on `98a22323`. PR-Agent reviewed, no findings beyond the verification. CodeRabbit's only comment was a "Review limit reached" notice: no review ran, disclosed in the triage; merged with one reviewer.
- **#3374 landed** with `dotf pr land` -> main `6057dd60`; #3372 closed, Done on the board; worktree and branch removed.
- Main on `ec724137`: `cli`, `CI`, `cli-smoke`, `Repo hygiene`, `release-please` green; `from-zero` still running. Main on `6057dd60`: `cli-smoke`, `Repo hygiene`, `release-please` green, `cli` (43150074590) and `CI` (43150074581) running. Its `from-zero` run (43150074675) was cancelled while queued, not by the `ec724137` run: the workflow's concurrency group on `main` has `cancel-in-progress` false, so it keeps one pending run and a newer push replaces it; #3371 (`8d01584f`) replaced this one and #3373 (`510a3a34`) replaced that, so the pending run on `510a3a34` (43150232030) is the one that covers the #3374 change.
