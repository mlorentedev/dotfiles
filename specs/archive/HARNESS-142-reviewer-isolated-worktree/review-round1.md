---
spec: "HARNESS-142-reviewer-isolated-worktree"
verdict: "FAIL"
reviewed_sha: "df6959540ea1ff9b9cab679e0686361aae2824a0"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-05"
---

## Adversarial review

**Scope**: HARNESS-142-reviewer-isolated-worktree
**Sources**: specs/HARNESS-142-reviewer-isolated-worktree/{proposal,tasks,verification}.md, git diff adf49c93bb006b6322bb9bc6b20f1a44303e16e9...HEAD

### Spec and task alignment
- **Proposal alignment**: The implementation cleanly fulfills the core goal of running the reviewer in a detached, disposable worktree (`AddDetached`) with metadata (`Kind: review`, `Issue: 0`, `ReapOK: true`, `LeaseExpiresAt`). The caller tree is guarded against dirty launches unless `--allow-dirty` is provided (AC5). Stray writes in the worktree are detected and exit 4 without leaking into the caller repository (AC1, AC2). Interrupted or expired review worktrees are reaped by `dotf worktree sweep` once their lease passes if clean or if `output_saved` is recorded (AC3).
- **Tasks alignment**: Implementation tasks across `cli/internal/worktree/`, `cli/internal/spec/`, and `cli/internal/cmd/` are implemented with unit and integration tests.
- **Contract set**: Contract digests match `review-request.json` without modification.
- **Defect discovered**: In multi-round reviews where `review.md` was previously committed at `HEAD`, git checkout in `AddDetached` populates `treeSpec/review.md` with the stale verdict from `HEAD`. Because `spec.RotateReviewRound` moves the caller's `review.md` to `review-round1.md`, `copySpecFolder` does not touch or delete `treeSpec/review.md`. If a review run subsequently aborts, crashes, or stalls before writing a new verdict, `finishReviewTree` finds `treeSpec/review.md` on disk and copies the old verdict back into the caller's tree, resurrecting the stale verdict and defeating the rotation guarantee.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | review-rotation | In a re-review where `review.md` was committed at `HEAD`, `createReviewTree` checks out `HEAD`'s `review.md` into `treeSpec/review.md`. Because `RotateReviewRound` renamed the caller's `review.md` to `review-round1.md`, `copySpecFolder` never deletes or overwrites `treeSpec/review.md`. If the reviewer stops, crashes, or stalls before writing, `finishReviewTree` copies `treeSpec/review.md` back to the caller, resurrecting the previous round's verdict and defeating rotation. | Observed via reproduction: creating a detached worktree at a commit holding `review.md` leaves `review.md` present alongside `review-round1.md`. `TestReviewWorktreeCarriesEarlierRounds` missed this because it stubbed `createReviewTree` with `os.MkdirAll` instead of checking out git. | UNTESTED | code (`cli/internal/cmd/spec.go`: remove `filepath.Join(treeSpec, slot.Review)` before reviewer launch) + tests (`cli/internal/cmd/spec_review_worktree_test.go`) |
| Minor | THEORETICAL | launch-cleanup | In foreground mode (`--foreground`), if `runForeground` fails before running `exec.Command` (e.g. `os.Create(transcript)` fails), `spec review` returns the error without removing the created review worktree. It remains until lease expiration reaps it via `sweep`. | Code review of `cli/internal/cmd/spec.go:292` and `cli/internal/cmd/spec.go:106`. | UNTESTED | code (`cli/internal/cmd/spec.go`) |
| Minor | THEORETICAL | handoff-readiness | `specs/HARNESS-142-reviewer-isolated-worktree/verification.md` has `Criterion 3` marked `(in progress)` and promotion candidate questions unanswered, which will block `dotf spec archive`. | `verification.md` lines 18, 49-51. | UNTESTED | spec (`verification.md` — outside contract set) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Core isolation and copy-back work as specified; edge case in committed review.md rotation prevents clean A. |
| Verification       | B | Excellent integration test coverage for stray writes and copy-back, but stubbed worktree creation masked rotation regression. |
| Scope              | A | Perfectly bounded to reviewer worktree isolation, caller dirt check, and sweep reaping; no scope creep. |
| Reliability        | B | Solid lease management, Gate f process checks, and sweep fallback; foreground launch abort leaves uncleaned worktree. |
| Maintainability    | A | Idiomatic Go, clear package separation, strict vet/lint clean across Darwin and Windows. |
| Handoff-readiness  | B | Features and evidence mapped, but verification.md has in-progress markers and unpopulated promotion candidates. |

### Verdict
FAIL

### Recommended next steps
1. **Fix stale review slot in review worktree**: In `cli/internal/cmd/spec.go` (inside `newSpecReviewCmd`), ensure that `filepath.Join(treeSpec, slot.Review)` is removed after `createReviewTree` and `copySpecFolder` before launching the reviewer, so that a committed `review.md` from `HEAD` cannot linger in the review worktree.
2. **Add regression test**: In `cli/internal/cmd/spec_review_worktree_test.go`, add a test with a real git repository where `review.md` was committed at `HEAD`, verifying that `filepath.Join(treeSpec, "review.md")` is absent in the worktree before the reviewer writes its verdict.
3. **Clean up `verification.md`**: Update `Criterion 3` status to complete and answer the promotion candidate questions in `specs/HARNESS-142-reviewer-isolated-worktree/verification.md`.
4. **Re-run adversarial review**: Once the fix is applied in code and verified with tests, trigger the follow-up review round.
