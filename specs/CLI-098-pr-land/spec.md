---
id: spec-CLI-098-pr-land
type: spec
status: implementing
created: "2026-10-06"
owner: manu
issue: "mlorentedev/dotfiles#2034"
---
# CLI-098: `dotf pr land`

## Intent

The owner authorises an agent to merge its own PRs when three conditions hold: CI green, reviewer output triaged (at least pr-agent), and no conflicts. Branch protection on `main` is `strict`, so each merge pushes the other open PRs BEHIND. Each of them then needs a rebase, a fresh CI run and often a fresh review before it can merge.

During the PLAT-001 macOS session, this check was done with a throwaway script in a scratch directory. Its first use showed why it should be a tested command: `--delete-branch` on a PR that another PR is stacked on made GitHub **close** the dependent PR (#2030). The three facts must also be read for **one head commit**, or a push between the reads can merge something nobody triaged.

## Scope

`dotf pr land <number> [--wait] [--repo owner/name] [--registry path]`:

1. With `--wait`, it first waits for the PR's checks (`gh pr checks --watch`).
2. It reads the PR's head SHA, then requires all of the following on that SHA:
   - the PR is open and not a draft;
   - its head branch is not a release-please branch (cutting a release is the owner's decision);
   - every check is `pass` or `skipping`;
   - `mergeStateStatus` is `CLEAN` (up to date with the base, no conflicts);
   - `dotf pr triage-queue` does not list it.
3. It re-reads the head SHA. If it moved, it refuses.
4. It retargets every PR whose base is this head branch to this PR's base.
5. It squash-merges with `--match-head-commit <sha> --delete-branch`.
6. Otherwise it exits 1 and names every condition that failed. It never uses `--auto`.

With `--update-branch`, when BEHIND is the only failing condition, it merges the base into the branch (`gh pr update-branch`: a merge, not a rebase, so the reviewer's push gate does not re-review and the squash flattens it anyway), waits for the new CI and decides again on the new head. Measured need: in the PLAT-001 merge chain on 2026-10-06, every merge into `main` from a parallel session left the next PR BEHIND, and each rebase re-triggered a full review. The update repeats while BEHIND stays the only reason, up to three times: on 2026-10-07 #2041 was updated, and a parallel session merged #2039 into `main` while the new head's CI ran, so one update left it BEHIND again and the queue stopped there. A base that keeps moving after three updates is a refusal that names BEHIND.

Out of scope: landing every PR of the user (`--all-mine`).

## Queue and lock (#2066)

`dotf pr land <n> [<n>...]` lands several PRs as one queue. Measured on 2026-10-07: four concurrent `pr land --wait --update-branch` processes against five PRs. Every merge marked the others BEHIND, every running process then pushed a merge commit of its own, and each push started a full CI run and a reviewer re-review. That is O(n^2) CI for n PRs, and with `maxUpdates = 3` the last PR in line can exhaust its updates and exit unmerged.

Scope:

1. The PRs land in the order given, one at a time. A PR is updated only during its own turn, so each PR pays one update per base move while it is the one landing.
2. A PR that stops for any reason (a failed check, DIRTY, reviewer output awaiting triage, an error reading a fact) is reported and the queue moves on. Only a cancelled context ends the queue; the PRs after it are reported as not attempted.
3. At the end one summary lists the merged PRs and, for each PR not merged, its reasons. It reuses `prland.Result`, which gains the PR number. The exit status is non-zero when any PR was not merged.
4. One PR keeps the output and the exit contract it had.
5. A per-repo lock under the state dir (`<state>/pr-land/<owner>+<name>.lock`, lowercased; `+` keeps `a-b/c` and `a/b-c` on separate locks). The lock is a kernel lock (`internal/filelock`), so the mutual exclusion has no race and a killed process cannot leave it held. A sibling `.pid` file records the holder. A second `pr land` that finds the lock held refuses and names the holder's PID. A `.pid` file left by a holder that is no longer alive is taken over with a note on stderr. The lock is released, and the `.pid` file removed, on every exit path.

Failing tests first (written before the code, run red, then green):

- `TestLandQueue_LandsInTheOrderGiven`, `TestLandQueue_UpdatesAPROnlyDuringItsOwnTurn`
- `TestLandQueue_ContinuesPastAStoppedPR`, `TestLandQueue_AnErrorStopsThatPRNotTheQueue`, `TestLandQueue_ACancelledContextLeavesTheRestNotAttempted`
- `TestAcquireLock_RefusesWhileHeldAndNamesThePID`, `TestAcquireLock_TakesOverAStaleLockWithANote`, `TestAcquireLock_ReleaseFreesTheLockAndRemovesThePIDFile`, `TestAcquireLock_LocksAreIndependentPerRepo`
- `TestPrLand_AQueueLandsInOrderAndSummarisesWithExitOne`, `TestPrLand_ASecondLandRefusesWhileTheLockIsHeld`, `TestPrLand_AStaleLockIsTakenOverWithANote`, `TestPrLand_ReleasesTheLockAfterAnError`, `TestPrLand_OnePRPrintsNoSummary`

## Checklist

- [x] Failing tests with a fake `gh` runner: lands when every condition holds; refuses and names each failed condition (a failing or pending check, BEHIND, DIRTY, untriaged review, draft, closed, release-please branch); refuses when the head moves between reads; retargets dependents before the merge; passes `--match-head-commit` with the SHA it checked
- [x] `internal/prland`: the decision as a pure function over the facts read, and the gh calls behind an injected runner
- [x] `cmd/pr.go`: `land` subcommand; `cli/README.md` row for `pr`
- [x] Verified on a real PR of this repository
- [x] Queue and lock (#2066): `LandQueue`, `AcquireLock`, `pr land <n> [<n>...]`, tests below, runbook `release-dotf.md`, lesson 344

## Evidence

- `go test ./internal/prland/ ./internal/cmd/ -count=1` -> ok (Decide: every failed condition named; Land: retarget-then-merge with `--match-head-commit`, refusal without mutation, untriaged refusal, moved head, unanswerable triage queue is an error; cmd: refusal output and exit 1, non-number rejected)
- Measured on #2025: `--wait` returned before a push's checks were registered, so the merge state read BLOCKED with every reported check green; the refusal was correct but premature. `--wait` now pauses and re-reads while a check is pending or the state is BLOCKED/UNKNOWN/UNSTABLE with every check green, up to 6 rounds (`TestLand_WaitReadsAgainUntilTheStateSettles`, `TestLand_WithoutWaitABlockedStateIsARefusal`)
- `--update-branch`: `TestLand_UpdateBranchMergesTheBaseThenLandsTheNewHead` (merges the base, then merges the new head with `--match-head-commit`), `TestLand_UpdateBranchDoesNotTouchAPRThatFailsForAnotherReason`, `TestLand_UpdateBranchUpdatesAgainWhenTheBaseMovesDuringTheWait`, `TestLand_UpdateBranchGivesUpAfterMaxUpdates`
- Transactional retarget (pr-agent on #2037): a merge that fails after the dependents were retargeted restores each to the head branch; a restore that fails is named in the error with the merge error (`TestLand_AFailedMergeRestoresTheRetargetedDependents`, `TestLand_AFailedRestoreNamesTheDependentLeftRetargeted`)
- Measured on #2033/#2037/#2039: right after a push gh reports no checks (non-zero exit, empty output), which `land` read as an error. Empty output is now "no checks reported" (a refusal) and keeps `--wait` waiting (`TestLand_NoChecksYetIsARefusalNotAnError`)
- Real PR, read-only: `dotf pr land 2033` -> `[NOT MERGED] #2033 at e656207: spec-gate: fail; merge state is UNKNOWN; reviewer output awaits triage`, exit 1, nothing changed
- Queue and lock (#2066): `go test ./internal/prland/ ./internal/cmd/` -> ok; mutation checks: reversing the loop in `LandQueue` fails all five `TestLandQueue_*`, dropping `defer release()` fails `TestPrLand_ReleasesTheLockAfterAnError` and `...AfterAQueue`; `GOOS=windows go vet ./...` and `GOOS=darwin go vet ./...` clean; `golangci-lint run` -> 0 issues

## Next

Land the first PR with it once a PR meets every condition.

## Promotion candidates

- [ ] Lesson? `no: the stacked-PR close is recorded on #2034 and handled by the command`
- [ ] ADR? `no`
- [ ] Pattern? `no`
