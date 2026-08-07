---
tags: [spec, tasks, templates]
created: "2026-08-05"
---

# Tasks - HARNESS-051-copilot-native-skills

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/copilot-native-skills`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] Write a failing smoke test that expects `spec` and `handoff` under `~/.copilot/skills`.
- [x] [AC2] Extend the smoke test for Copilot target filtering and auxiliary files.
- [x] [AC3] Add a convergence test proving generated stale skills are pruned without deleting user-managed skills.
- [x] [AC1] Add the native Copilot skill target to `harness/manifest.json`.
- [x] [AC1] Update both Copilot instruction overlays to describe native discovery accurately.
- [x] [AC4] Deploy to an isolated home and confirm `copilot skill list` recognizes `handoff`.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Harness syntax and drift checks pass
- [x] Targeted Bats tests pass
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

See the sibling `features.json`.
