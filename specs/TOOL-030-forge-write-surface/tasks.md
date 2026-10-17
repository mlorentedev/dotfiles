---
tags: [spec, tasks, templates]
created: "2026-10-16"
---

# Tasks - TOOL-030-forge-write-surface

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [ ] Branch created from main for each PR below (the spec PR itself is `docs/tool-030-forge-write-surface`)
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] The open question about the classifier is answered by PR 1, not assumed

## Implementation

> Four PRs, sequenced. PR 1 measures before anything widens authority.

**PR 1: the interface and the measurement (no allow rules)**
- [ ] [AC1] [AC3] Write failing tests for forge detection from origin and for body-from-file only
- [ ] [AC1] [AC3] Add `cli/internal/forgewrite` with the `Forge` interface, detection (reusing `initrepo.ClassifyOrigin`) and the GitHub backend over the `gh` REST endpoint
- [ ] [AC4] Add the Gitea REST backend; the token comes from `dotf secrets` at call time; test that it never leaks
- [ ] [AC1] Wire `dotf forge issue create` and `issue comment`
- [ ] [AC6] Run one Gitea comment from an agent session and record the classifier's answer on #2635

**PR 2: the remaining verbs**
- [ ] [P] [AC2] `issue edit`, `pr create`, `pr comment`, `pr edit`, each with a failing test first
- [ ] [AC2] Test that no `merge` or `close` verb is registered
- [ ] `--board` for GitHub Project fields, degrading with a stated message on Gitea

**PR 3: the guard**
- [ ] [AC5] `tests/harness-permits-dotf.bats`, one `@test` per harness config, passing under bash 3.2 and zsh

**PR 4: skills and the allow set (owner lands)**
- [ ] [AC7] `new-ticket`, `pr-review-triage` and `autopilot` call `dotf forge` (vault edit, then refresh the records)
- [ ] [AC7] Settings PR: the four `dotf forge` allows and the revert of #2629; the owner lands it (#2178)

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

Minimal `features.json` skeleton (drop into `<repo>/specs/TOOL-030-forge-write-surface/features.json`):

```json
[
  {
    "id": "TOOL-030-forge-write-surface-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
