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
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

Worktree package (`cli/internal/worktree/`):

- [ ] [P] [AC3] Test: `AddDetached` creates a worktree detached at a given sha, with metadata `Issue: 0`, `ReapOK: true`, a lease of the given TTL and `Kind: "review"`; fails red because `AddRunner` has no detached path. `go test ./internal/worktree/ -run AddDetached`
- [ ] [AC3] Add `WorktreeAddDetached(repo, target, sha)` to `AddRunner` and `RealAddRunner`, a `Kind` field to `Metadata`, and `AddDetached`; test green.
- [ ] [AC3] Test then code: `dotf worktree list` labels a `Kind: "review"` worktree, and `refuseHeldIssue` never counts it (it holds no issue).
- [ ] [AC3] Test then code: `sweep` removes an expired review worktree whose `ReapOK` is true.

Launch (`cli/internal/cmd/spec.go`, `cli/internal/spec/`):

- [ ] [P] [AC5] Test then code: `spec review` refuses on tracked dirt (via `worktree.Dirt`) and names `--allow-dirty`; with the flag it launches and warns.
- [ ] [AC4] Test: the launch path builds the agy argv with `--add-dir <worktree>`, `TmuxWrap` with the worktree as its start directory, and the prompt naming the fresh worktree at `<sha>`; red against today's repoRoot. Enumerate every `filepath.Join(repoRoot, "specs", id, ...)` on the launch path and assign it a root: reviewer-side (review file, transcript, `.stderr`, the `--review` and `--transcript` flags to `deadline`, the stray-write snapshot) is the worktree; caller-side (`review-request.json`, round rotation, the copy-back destination) is the caller's root.
- [ ] [AC4] Implement: the launcher creates the review worktree (name: spec id, slot, nonce; TTL: deadline plus margin) and passes it and the caller's spec folder to `deadline`; test green.
- [ ] [AC6] Test then code: before launch, copy the caller's `specs/<id>/` review history into the worktree's spec folder.
- [ ] [AC4] Prompt sentence: fresh worktree at `<sha>`, no build artifacts, install what the suite needs first. Correct the `agyReviewerCommand` comment.

Deadline (`spec deadline`):

- [ ] [AC1] [AC3] Test: with a fake reviewer, outputs are copied back and the worktree removed on exit 0, 4, 124 and 125; red first.
- [ ] [AC1] [AC3] Implement copy-back and removal on every exit path; a failed removal is reported on stderr and left for `sweep`.
- [ ] [AC1] [AC2] Integration test: a fake reviewer creates `tests/probe_test.go` and edits a tracked file; the caller's `git status --porcelain` is unchanged, `review.md` and the transcript arrive, both paths are named and the run exits 4.

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
