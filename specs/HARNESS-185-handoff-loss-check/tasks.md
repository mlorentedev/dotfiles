---
tags: [spec, tasks, templates]
created: "2026-11-01"
---

# Tasks - HARNESS-185-handoff-loss-check

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [ ] Branch created from main: `spec/1881-handoff-loss`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [ ] [P] [AC1] Failing test: a linked worktree, `threads/<key>.md` present, no entry -> WARN naming key and file
- [ ] [AC1] `checkHandoffThreads` in `cli/internal/doctor/checks_handoff_threads.go`: list worktrees, key each with `mem.ThreadKey`, read `threads/`, scan `MEMORY.md` headings; register it after "Auto-memory vault link"
- [ ] [P] [AC2] Failing test: entry present, and a `<key>+<agent>` fork entry -> pass
- [ ] [P] [AC3] Failing test: file moved to `archive/` -> no warning
- [ ] [P] [AC4] Failing test: no `threads/<key>.md` but a journal under `sessions/` -> no warning
- [ ] [AC5] Skips with their reasons: no vault, no vault project, no `MEMORY.md`, no `threads/` (the last names the inline layout as not covered)
- [ ] [AC6] Test that the vault tree is byte-identical after the check and after `--fix`, and that no path in the check can call `rep.Fail`
- [ ] [AC7] Test with a detached worktree and a branch containing `/`; the expected key is whatever `mem.ThreadKey` returns for the same path
- [ ] [AC8] Run once against the real vault, record which entry-less files belong to a live worktree in `verification.md`
- [ ] Runbook line: where the check is described next to the handoff docs (`docs/runbooks/`), including the archive step as the second remedy

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

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-185-handoff-loss-check/features.json`):

```json
[
  {
    "id": "HARNESS-185-handoff-loss-check-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
