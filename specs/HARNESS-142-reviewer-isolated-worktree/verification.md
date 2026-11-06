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

- [ ] Criterion 1 -> commit `<hash>` / test `<name>`
- [ ] Criterion 2 -> commit `<hash>` / test `<name>`
- [ ] Criterion 3 -> (in progress) worktree package: `TestAddDetachedSavesALeaseFreeReapableLease`, `TestAddWithIssueIgnoresALiveReviewWorktree`, `TestWorktreeTableLabelsAReviewWorktree` (#3195); `TestClassifyAReviewWorktreeByItsLeaseAndSavedOutput`, `TestReviewWorktreeSwept`, `TestSweepKeepsAReviewWorktreeThatMayHoldTheOnlyVerdict`, `TestSweepKeepsAReviewWorktreeWhileItsLeaseLivesOrAReviewerRuns`, `TestRealRunnerForceRemovesAReviewWorktreeWithUntrackedOutput`; copy-back and removal on every exit path are still open

## Test status

- Test suite: `cd cli && go test ./internal/worktree/ ./internal/cmd/ -count=1` -> ok (worktree package batch; the full suite runs at the end per tasks.md)
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- A review worktree is reaped on its lease, once its output is safe. Under the ordinary gates sweep could never take one: it has no branch, so no pull request merges it, and an interrupted review leaves its output untracked, so it is never clean. AC3 was unreachable until `Classify` and `reapSingleWorktree` keyed `Kind: review` on the lease, skipping the clean and merged gates. The cwd check and Gate f still refuse, so a reviewer running past its lease is kept. Sweep removes it with `git worktree remove --force` (plain remove refuses untracked files) and says on stderr that it discarded what else the reviewer left.
- The lease alone was not enough (review of #3196): a forced removal of a dirty review worktree can delete the only copy of a verdict when `spec deadline` failed before copy-back, and a failed `git status` read as dirt. Sweep now reaps a review worktree only when it is clean or `output_saved` is set, which `spec deadline` writes after copy-back and before removal; an unreadable state keeps it with a warning. `Classify` lists the unsaved case as DIRTY with the manual recovery.
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
