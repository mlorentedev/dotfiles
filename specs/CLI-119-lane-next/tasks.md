---
tags: [spec, tasks, templates]
created: "2026-10-14"
---

# Tasks - CLI-119-lane-next

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/lane-next` (herdr worktree `feat-lane-next`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions" (the pane-readiness wait and the Windows exec are measured in PR 2, not assumed)

## Implementation

Two PRs, each under the ~300 executable-line cap. PR 1 is everything that needs no terminal; PR 2 adds the herdr backend and the real runs.

PR 1 - `internal/lane`: launch table, queue, prompt, `dotf lane exec`

- [ ] [P] [AC1] Failing test `TestLaunchArgv_PinsEachAgentAndPassesThePromptAsOneArgument`: claude, pi and agy argv, a multi-line prompt with quotes stays one element
- [ ] [AC1] `lane.LaunchArgv(agent, prompt)` and the table; unknown agent returns an error naming the known ones
- [ ] [P] [AC3] Failing test `TestQueue_NextIsTheFirstOpenUnassignedIssueAndIsStable` with a fake issue source
- [ ] [AC3] `lane.Config` (`.dotf/lane.json`), `lane.Next`, slug function, prompt render from the template
- [ ] [AC1] `dotf lane exec --agent --prompt-file` (Unix: `syscall.Exec`; other: child with stdio), registered in root

PR 2 - `dotf lane next` and the herdr backend

- [ ] [P] [AC2] Failing test `TestNext_FromALinkedWorktreeCreatesThroughTheParentAndLeavesTheCallerAlone` with a recording backend
- [ ] [AC2] [AC3] `lane.Backend` interface, `lane.Next` orchestration, reuse of an existing worktree
- [ ] [P] [AC4] Failing tests `TestNext_AgentDefaultsToTheCallingPaneAndUnknownIsRefused`
- [ ] [AC5] Failing test `TestNext_NoBackendPrintsTheCommands`
- [ ] herdr backend: `worktree create --cwd <parent>`, `pane run`, a `wait-output` for the shell prompt (measure first)
- [ ] [AC6] Real runs: finisher -> successor for claude, pi and agy; paste the evidence into `verification.md`

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

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-119-lane-next/features.json`):

```json
[
  {
    "id": "CLI-119-lane-next-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
