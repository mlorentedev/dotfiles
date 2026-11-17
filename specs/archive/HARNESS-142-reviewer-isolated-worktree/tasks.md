---
tags: [spec, tasks, templates]
created: "2026-11-04"
id: "HARNESS-142-reviewer-isolated-worktree"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-142-reviewer-isolated-worktree

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `spec/reviewer-isolated-worktree`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

Worktree package (`cli/internal/worktree/`):

- [x] [P] [AC3] Test: `AddDetached` creates a worktree detached at a given sha, with metadata `Issue: 0`, `ReapOK: true`, a lease of the given TTL and `Kind: "review"`; fails red because `AddRunner` has no detached path. `go test ./internal/worktree/ -run AddDetached`
- [x] [AC3] Add `WorktreeAddDetached(repo, target, sha)` to `AddRunner` and `RealAddRunner`, a `Kind` field to `Metadata`, and `AddDetached`; test green.
- [x] [AC3] Test then code: `dotf worktree list` labels a `Kind: "review"` worktree, and `refuseHeldIssue` never counts it (it holds no issue).
- [x] [AC3] Test then code (`TestReviewWorktreeSwept`): `sweep` removes an expired review worktree that is clean or whose output was copied back (`OutputSaved`), and keeps one whose output was not, or whose state cannot be read.

Launch (`cli/internal/cmd/spec.go`, `cli/internal/spec/`):

- [x] [P] [AC5] Test then code: `spec review` refuses on a tracked change or untracked file outside `specs/<id>/` (a non-empty `spec.SnapshotTree`, which already leaves the spec folder out), naming the paths and `--allow-dirty`; with the flag it launches and warns; dirt inside `specs/<id>/` launches.
Launch and deadline land together (amended while implementing): a launch into the worktree without the copy-back would rotate the previous verdict out and leave the new one in a worktree only, so every review on `main` between the two PRs would strand its verdict.

- [x] [AC4] Test (`TestReviewLaunchUsesReviewWorktree`): the agy argv carries `--add-dir <worktree>`, tmux starts in it, the prompt names it as a fresh worktree at `<sha>`, and `deadline` gets `--review` in the worktree plus `--worktree` and `--copy-back`. Reviewer-side: the review file, `--review`, the stray-write snapshot. Caller-side: `review-request.json`, round rotation, the copy-back destination, and the transcript and `.stderr`, which the launcher's pipeline writes outside the deadline process.
- [x] [AC4] Implement: the launcher plans the review worktree (name: spec id, slot, nonce; TTL: deadline plus 30 minutes), creates it after the dry-run return, and removes it again when the request, the copy-in or the tmux launch fails (`TestReviewLaunchRemovesTheWorktreeWhenTmuxFails`).
- [x] [AC6] Test then code (`TestReviewWorktreeCarriesEarlierRounds`): after round rotation, copy the caller's uncommitted `specs/<id>/` into the worktree; a second first-signer round finds `review-round1.md`.
- [x] [AC4] Prompt sentence (`spec.FreshWorktreeNote`): fresh worktree at `<sha>`, no build artifacts, install what the suite needs first. The `agyReviewerCommand` comment already says the reviewer runs in an isolated worktree, which is now true.
- [x] [AC1] [AC3] Test (`TestDeadlineCopiesBackAndRemovesReviewWorktree`): with a fake reviewer, the slot's review file is copied back and the worktree removed on exit 0, 4, 3, 124 and 125, and for `--second` (`review-second.md`).
- [x] [AC1] [AC3] Implement copy-back and removal on every exit path; a failed removal is reported on stderr and left for `sweep`; a failed copy-back keeps the worktree (`TestDeadlineKeepsTheWorktreeWhenTheCopyBackFails`).
- [x] [AC3] After copy-back succeeds and before removal, set `OutputSaved` in the worktree's metadata.
- [x] [AC1] [AC2] Integration tests (`TestReviewInWorktreeLeavesCallerTreeUntouched`, `TestReviewInWorktreeStillReportsStrayWrites`): a fake reviewer creates `tests/probe_test.go` and edits a tracked file in a real review worktree; the caller's status shows only the copied-back verdict, both paths are named and the run exits 4.

Verification of the whole:

- [ ] `go build ./... && go vet ./... && go test ./...`, `golangci-lint run`, `GOOS=windows go vet ./...`, `GOOS=windows golangci-lint run ./...` from `cli/`.
- [ ] One real `dotf spec review` on this spec, recorded in `verification.md`: the caller tree is untouched and no review worktree is left.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-142-reviewer-isolated-worktree/features.json`):

```json
[
  {
    "id": "HARNESS-142-reviewer-isolated-worktree-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
