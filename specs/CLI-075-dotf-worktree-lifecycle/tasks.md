---
tags: [spec, tasks, templates]
created: "2026-09-04"
---

# Tasks - CLI-075-dotf-worktree-lifecycle

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Work-gate OK: mlorentedev/dotfiles#1500 is open and assigned
- [x] `proposal.md` is complete and acceptance criteria are testable (AC1..AC7)
- [x] Adversarial review Round 1 & Round 2 addressed in design

## PR Sequence

| PR | Scope | Criteria |
|---|---|---|
| **PR 1** | Discovery & State Machine: `dotf worktree list` | AC1, AC2 |
| **PR 2** | Standardized Creation: `dotf worktree add` | AC3 |
| **PR 3** | Fail-Closed Reaper & Teardown: `dotf worktree sweep` & `done` | AC4, AC5, AC6, AC7 |

## Implementation

### PR 1 — Discovery & State Machine (`dotf worktree list`)

- [x] [P] [AC1] Test: Parse `git worktree list --porcelain` into typed Worktree structs.
- [x] [AC1] Implement `worktree.List()` parsing paths, heads, and branch names.
- [x] [P] [AC2] Test: Ignore submodule checkouts where `gitdir` contains `modules/`.
- [x] [AC2] Implement submodule filtering in `worktree.List()`.
- [x] [P] [AC1] Test: Read `.dotf-worktree.json` lease metadata and detect dirty git status.
- [x] [AC1] Implement metadata loading and `git status --porcelain` classification (`ACTIVE`, `REAPABLE`, `DIRTY`, `UNMERGED`).
- [x] [AC1] Wire `dotf worktree list` Cobra command with text table and `--json` support.

### PR 2 — Standardized Creation (`dotf worktree add`)

- [x] [P] [AC3] Test: Verify external sibling path resolution (`<repo>-wt-<slug>`) outside parent repo.
- [x] [AC3] Implement path validation rejecting nested or in-repo worktree paths.
- [x] [P] [AC3] Test: Worktree creation writes `.dotf-worktree.json` and excludes it in `.git/info/exclude`.
- [x] [AC3] Implement `worktree.Add()` executing `git worktree add`, writing metadata, and setting lease.
- [x] [AC3] Wire `dotf worktree add <slug> [--issue <N>] [--ttl <duration>]` Cobra command.

### PR 3 — Fail-Closed Reaper & Teardown (`dotf worktree sweep` & `done`)

- [x] [P] [AC6] Test: Cross-process lock (`lock_unix.go` / `lock_windows.go`) blocks concurrent sweeps.
- [x] [AC6] Implement cross-platform file locking for sweep operations.
- [x] [P] [AC4] Test: Sweep refuses dirty, unmerged, lease-active, or young (<15m) worktrees.
- [x] [AC4] Implement fail-closed evaluation logic for candidate worktrees.
- [x] [P] [AC5] Test: Reaping removes worktree, logs SHA to stderr before branch deletion, and prunes git metadata.
- [x] [AC5] Implement safe reap execution with stderr SHA logging.
- [x] [P] [AC7] Test: `dotf worktree done` cleanly tears down current worktree when pushed.
- [x] [AC7] Implement `dotf worktree done` command.
- [x] [AC4] Wire `dotf worktree sweep [--dry-run]` Cobra command.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks and tests pass (`go -C cli test ./...`)
- [x] No unrelated changes in diff
- [x] `verification.md` filled in with command outputs
- [ ] Adversarial review passes before archive (`dotf spec review CLI-075-dotf-worktree-lifecycle`)
