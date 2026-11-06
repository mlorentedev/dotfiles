---
tags: [spec, verification, templates]
created: "2026-11-04"
id: "HARNESS-142-reviewer-isolated-worktree"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-142-reviewer-isolated-worktree

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> `TestReviewInWorktreeLeavesCallerTreeUntouched` (real review worktree, fake reviewer: the caller's status shows only the copied-back verdict)
- [x] Criterion 2 -> `TestReviewInWorktreeStillReportsStrayWrites` (both paths named, exit 4)
- [x] Criterion 3 -> worktree package: `TestAddDetachedSavesALeaseFreeReapableLease`, `TestAddWithIssueIgnoresALiveReviewWorktree`, `TestWorktreeTableLabelsAReviewWorktree` (#3195); `TestClassifyAReviewWorktreeByItsLeaseAndSavedOutput`, `TestReviewWorktreeSwept`, `TestSweepKeepsAReviewWorktreeThatMayHoldTheOnlyVerdict`, `TestSweepKeepsAReviewWorktreeWhileItsLeaseLivesOrAReviewerRuns`, `TestRealRunnerForceRemovesAReviewWorktreeWithUntrackedOutput`; copy-back and removal on every exit path: `TestDeadlineCopiesBackAndRemovesReviewWorktree` (exit 0, 4, 3, 124, 125 and `--second`), `TestDeadlineKeepsTheWorktreeWhenTheCopyBackFails`
- [x] Criterion 4 -> `TestReviewLaunchUsesReviewWorktree` (agy and pi: tmux start directory, prompt, `--add-dir` for agy, the deadline flags), `TestReviewLaunchRemovesTheWorktreeWhenTmuxFails`
- [x] Criterion 5 -> `TestReviewRefusesDirtyTreeWithoutAllowDirty` (real repository: dirt inside `specs/<id>/` launches, an untracked file outside refuses naming it and `--allow-dirty`, the flag launches and warns), `TestCallerDirtNamesWhatTheReviewerWouldNotSee`
- [x] Criterion 6 -> `TestReviewWorktreeCarriesEarlierRounds` (`review-round1.md`, both requests and the uncommitted `proposal.md` reach the worktree), `TestReviewWorktreeDropsAVerdictTheCallerRotated` (a verdict committed at HEAD does not survive the rotation in the worktree)

## Test status

- Worktree package batch (#3195, #3196): `cd cli && go test ./internal/worktree/ ./internal/cmd/ -count=1` -> ok.
- Launch batch, AC5: `cd cli && go build ./... && go vet ./... && go test ./...` -> every package ok; `golangci-lint run ./...`, `GOOS=windows golangci-lint run ./...` -> 0 issues; `GOOS=windows go vet ./...` -> clean.
- The command-level review fixtures use a fake `.git` directory, so they stub the dirt check through the `callerDirt` seam, as they already stub the review base. `TestReviewRefusesDirtyTreeWithoutAllowDirty` replaces the fake with a real repository and runs the real check, and `TestCallerDirtNamesWhatTheReviewerWouldNotSee` covers the helper in `internal/spec`.
- Acceptance run, 2026-11-05 20:51-21:04 MST: `dotf spec review HARNESS-142-reviewer-isolated-worktree --reviewer agy/gemini-3.8-flash-high --timeout 30m` from a binary built off df695954, so the sandboxed agy arm is the one exercised. Caller `git status --short` before: empty. The worktree `harness-142-reviewer-isolated-worktree-first-0f70fa` was created detached at df695954; inside agy's sandbox (`--add-dir` naming only the worktree) `git rev-parse HEAD` and the diff commands succeeded, so the linked worktree reaches its common git dir. The run ended on its own after 12 minutes, `review.md` was copied back, and `git worktree list` no longer lists the worktree, nor is its directory on disk. Caller status after: only `specs/HARNESS-142-reviewer-isolated-worktree/review.md` and `review-request.json`, both untracked (the transcript is git-ignored). The `.stderr` holds one `mise WARN Current directory does not exist or is not accessible` line and nothing else.
- That run's verdict was FAIL, on a real defect: a verdict committed at HEAD was checked out into the worktree and survived the caller's rotation, so a run ending without a verdict would copy the old one back. Fixed by mirroring the caller's spec folder instead of copying over it (`TestReviewWorktreeDropsAVerdictTheCallerRotated`). Its minor finding, a foreground run that never starts leaving the worktree to its lease, is fixed too (`TestReviewForegroundRemovesTheWorktreeWhenTheRunNeverStarts`).
- Launch and deadline batch (AC1, AC2, AC3, AC4, AC6): the full Go suite, both lints and `GOOS=windows go vet ./...` pass; the deadline tests run real git worktrees and are Unix-only, like the stray-write tests they extend.
- Not yet exercised for real, left to the archive review: a pi reviewer in a review worktree, and the worktree's removal on a real Windows box (CI's Windows leg runs the unit tests only).
- No regressions in the existing suite: yes, the full Go suite passes with the check in place.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- A review worktree is reaped on its lease, once its output is safe. Under the ordinary gates sweep could never take one: it has no branch, so no pull request merges it, and an interrupted review leaves its output untracked, so it is never clean. AC3 was unreachable until `Classify` and `reapSingleWorktree` keyed `Kind: review` on the lease, skipping the clean and merged gates. The cwd check and Gate f still refuse, so a reviewer running past its lease is kept. Sweep removes it with `git worktree remove --force` (plain remove refuses untracked files) and says on stderr that it discarded what else the reviewer left.
- The lease alone was not enough (review of #3196): a forced removal of a dirty review worktree can delete the only copy of a verdict when `spec deadline` failed before copy-back, and a failed `git status` read as dirt. Sweep now reaps a review worktree only when it is clean or `output_saved` is set, which `spec deadline` writes after copy-back and before removal; an unreadable state keeps it with a warning. `Classify` lists the unsaved case as DIRTY with the manual recovery.
- The dirt check runs before `--dry-run` prints, so a dry run shows the refusal a real launch would hit. A tree whose state `git status` cannot read is refused as well, since it cannot be shown clean; `--allow-dirty` passes both.
- Launch and deadline ship in one PR, against the two batches tasks.md first drew: rotation empties the caller's slot, so a launch into the worktree with no copy-back would strand every verdict on `main` until the second PR.
- The transcript and its `.stderr` stay in the caller's spec folder rather than the worktree. In tmux mode the transcript sink is a separate process piped from `deadline`, still flushing when `deadline` removes the worktree, and on Windows its open file would block the removal. `confirmLaunched` and `foregroundOutcome` already read them caller-side. Only the review file is copied back.
- A failed copy-back keeps the worktree and leaves `output_saved` unset, so sweep keeps it too: it holds the only copy of the verdict.
- `deadline` leaves the worktree before removing it when its working directory is inside it, since Windows refuses to remove the working directory of a running process.
- The worktree's spec folder mirrors the caller's: files the caller does not hold are deleted after the copy. A plain copy kept whatever HEAD had committed there, which is how a rotated verdict came back (review round 1).
- `DetachedOptions.Kind` is required: a detached worktree with no issue and no kind is one nothing labels and no reap rule recognises (PR-Agent on #3195).

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-142-reviewer-isolated-worktree/` -> `specs/archive/HARNESS-142-reviewer-isolated-worktree/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
