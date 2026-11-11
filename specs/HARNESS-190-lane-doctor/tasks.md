---
tags: [spec, tasks, templates]
created: "2026-11-11"
id: "HARNESS-190-lane-doctor"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-190-lane-doctor

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created: `feat/3262-lane-doctor`, stacked on `feat/3261-lane-identity` until #3325 lands
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"
- [x] Measured on a scratch repository what git keeps after `mv`, `git worktree repair` and `git worktree move` (proposal, "What")

## Implementation

- [x] [AC1] Write failing tests in `cli/internal/lane/audit_test.go`: `AuditLanes` on a coherent repository (no findings), an unmanaged one, a marker with no lane present, and each corrupted-marker shape naming its field (396b590a)
- [x] [AC1] Implement `AuditLanes`, `Finding` and the marker validation in `cli/internal/lane/audit.go` to make them pass (ec8d58a7)
- [x] [AC2] [AC3] Write failing tests: `mv`'d lane, `git worktree move`'d lane, undeclared record, deleted lease, deleted record, lease naming another lane, plain directory at a lane path (396b590a, one red commit for the three criteria)
- [x] [AC2] [AC3] Implement the admin-dir enumeration and the per-lane checks to make them pass (ec8d58a7)
- [x] [AC4] Write failing tests in `cli/internal/cmd/lane_doctor_test.go`: silent exit 0 on a coherent repository, one line on an unmanaged one, exit 1 with ERROR/FIX pairs and no `Error:` line on a finding (48ad8029)
- [x] [AC4] Implement `dotf lane doctor` in `cli/internal/cmd/lane_doctor.go` (SilenceErrors, `withExitCode(1, …)`) (e2bcd9d7)
- [x] Runbook: `dotf lane doctor` section 7.6 and the known limit on the lane-local marker copy; lesson 581 on enumerating git's own registry rather than paths
- [x] File the follow-up ticket for the `dotf doctor` sweep entry (two options): #3327

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.
