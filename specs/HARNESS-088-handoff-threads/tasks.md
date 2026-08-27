---
tags: [spec, tasks, templates]
created: "2026-08-27"
---

# Tasks - HARNESS-088-handoff-threads

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [ ] Branch created from main: `feat/HARNESS-088-handoff-threads`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> Replace these with the actual steps for this feature. Keep them small (one commit each) and in TDD order.
> The `[P]` / `[AC<n>]` markers are optional — see the legend above. Behaviors 1 and 2 below are independent, so their *first* test task carries `[P]`.

- [ ] [P] [AC1] Write failing test for <behavior 1>
- [ ] [AC1] Implement <module/function> to make it pass
- [ ] Refactor for clarity (extract, rename, dedupe)
- [ ] [P] [AC2] Write failing test for <behavior 2>
- [ ] [AC2] Implement to make it pass
- [ ] ...

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

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-088-handoff-threads/features.json`):

```json
[
  {
    "id": "HARNESS-088-handoff-threads-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```

## Found by audit, fixed in this PR

- [x] The thread key asked a naming convention instead of git, so any worktree
      made by another tool (Claude Code's `.claude/worktrees/`, Orca's) resolved
      to `main` and every such session would have clobbered the others — the bug
      itself, reintroduced for anyone not using this repo's convention.
- [x] `SessionEnd` archived every worktree to one hardcoded `<date>-<project>-claude.md`
      with a **truncating** write, so the last session destroyed the others'
      durable records, and it assembled a filename the skill did not share.
- [x] Both above written test-first and mutation-checked.

## The debt, now fixed

- [x] **`SessionEnd` resolved the project as `filepath.Base(cwd)`** — a session in
      a subdirectory (`cli/`, where most work here happens) resolved the wrong
      project, found no `MEMORY.md`, and **silently archived nothing**. git is
      authoritative now; the basename stays as a fallback so a session outside any
      repository keeps working, because fixing a defect must not quietly narrow
      who the function serves.
- [x] **A thread is the BRANCH, not the worktree.** The vault is the SSOT across
      machines, so `feat/x` continued on a second box must resolve to the SAME
      thread rather than forking one. The hostname qualifies only `main`/`master`
      — ambient work rather than a line of it — and a detached HEAD is named
      `<worktree>@<host>` rather than collapsing into a plausible `main`.
- [x] **One `RepoIdentity` feeds both.** Two derivations of one fact in one
      package would have been the sixth divergent-parser defect found here in a
      week, this time between my own functions.

## Found by audit, NOT fixed — recorded rather than folded in

- [ ] ~~**`SessionEnd` resolves the project as `filepath.Base(cwd)`.**~~ Fixed
      above. Original note: A session
      whose working directory is a subdirectory — `cli/`, where most work in this
      repository happens — resolves the wrong project name, finds no `MEMORY.md`,
      and **silently archives nothing**. Same class as the thread-key defect, in
      a different function, and it means an unknown number of past sessions may
      have produced no archive at all. Left out because it changes *which*
      sessions archive, which needs its own measurement and its own evidence, not
      a fold-in at the end of another PR.
- [ ] **The gate's session state is keyed by session id alone** (`~/.local/state/dotfiles/gate/`).
      Two harnesses reusing an id (`s1` from pi and `s1` from opencode) would
      share a consumption record. Low exposure today — real harnesses send UUIDs —
      and the same argument the reviewer's collision finding won on #1275, so it
      should be fixed the same way when touched.
