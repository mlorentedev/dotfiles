---
tags: [spec, tasks, templates]
created: "2026-10-15"
---

# Tasks - CLI-127-lane-reconcile

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/2550-lane-reconcile` (worktree `feat-2550-lane-reconcile`)
- [x] `proposal.md` is complete and acceptance criteria are testable: owner approved the draft on 2026-10-16
- [x] No open questions left in `proposal.md` "Risks / open questions": the two MUST items became D1 and D2, and D3 was added (2026-10-16)
- [x] `/spec check CLI-127-lane-reconcile` returns PASS (2026-10-16: AC1 5 tasks, AC2 3, AC3 3, AC4 2, all by `[AC<n>]` tag; no orphans)

## Implementation

> Three PRs, one per verb. All code goes in `cli/internal/lane/` and `cli/internal/cmd/lane.go`. `cli/internal/worktree` and `cli/internal/herdr` are reached through their exported API only, because CLI-118 holds them. Run every test from `cli/`.

### PR 1: `dotf lane status`

- [x] [AC1] Write `cli/internal/lane/phase_test.go`: a table test over `Derive(Facts) Phase`, one row per phase (`Pending`, `Running`, `Review`, `Landing`, `Landed`, `Reaped`, `Stalled`, `Orphan`, `Unadopted`). `Facts` holds a `worktree.Info`, the PR state, `Live bool` and `HerdrPresent bool`. At least one row has herdr absent. Run `go test ./internal/lane/ -run TestDerive`; expected: FAIL, `undefined: Derive`.
- [x] [AC1] Write `cli/internal/lane/phase.go` with `Phase`, `Facts` and `Derive` until the test passes. Phases are computed and never stored.
- [ ] [AC1] Write `cli/internal/lane/status_test.go` (`TestStatus`). `Status(Sources)` joins fakes for `worktree.ListWithRunner`, a forge seam (`PRFor(branch)`, one REST request per run, cached), liveness (`worktree.RecentSession` plus the process seam) and herdr's `Find`. It returns one `Row` per worktree. The test asserts the rows and that the forge seam is asked once per branch. Expected: FAIL.
- [ ] [AC1] Write `cli/internal/lane/status.go` until it passes. If herdr is unavailable, rows are still returned, with `herdr: absent`.
- [ ] [AC1] In `cli/internal/cmd/lane.go`, add `newLaneStatusCmd` (`--json`) with seams like `laneExec`/`laneRun`, plus `cli/internal/cmd/lane_status_test.go` covering the text and JSON shapes and the exit code.
- [ ] Verify: `go build ./... && go vet ./... && go test ./...`, `GOOS=windows go vet ./...`, and `golangci-lint run` at the pin. Lesson. Open the PR.

### PR 2: `dotf lane plan`

- [ ] [AC2] Write `cli/internal/lane/plan_test.go` (`TestPlan`): a table test over `Plan(rows, queue, now) []Action`. Each `Action` has `Kind` (reap, start, adopt, flag), `Class` (`auto`, `pending`, `ambiguous`), `Due` for `pending` and `Resolve` (the exact command) for `ambiguous`. There is one row per D1 case, a D2 row with a derivable branch (adopt offered as `ambiguous`), a D2 row without one (no proposal), and a D3 row (landing order printed, no land action). Expected: FAIL.
- [ ] [AC2] Write `cli/internal/lane/plan.go` until it passes.
- [ ] [AC2] Add `newLanePlanCmd` and `TestPlanMutatesNothing` in `cli/internal/lane/`, asserting that the fixture repo's files and refs are byte-identical after `plan`, using a hash of `git for-each-ref` plus the lease files, before and after.
- [ ] Verify (same suite as PR 1). Lesson. Open the PR.

### PR 3: `dotf lane apply`

- [ ] [AC3] Write `cli/internal/lane/apply_test.go` (`TestApply`): `Apply(actions, verbs)` calls `verbs.Sweep` and `verbs.Next` only for `auto` actions, never adopt or land (asserted on a recording fake), and returns `changed`. A second `Apply` over the post-state returns `changed=0`. Expected: FAIL.
- [ ] [AC3] Write `cli/internal/lane/apply.go` until it passes. Reaps go through `worktree.Sweep` (its fail-closed gates, the process gate f included, are the safety). Starts go through the existing `lane next` path.
- [ ] [AC4] Add `TestApplyNeverReapsALiveWorktree` with a row where the process seam reports a process inside a merged, clean, expired worktree, and another where `RecentSession` reports a recent transcript. Neither is reaped, and both show as `ambiguous`.
- [ ] [AC3] Add `newLaneApplyCmd` printing `changed=N`, plus its cmd test.
- [ ] [AC4] Live run on this machine, read-only verbs only: `dotf lane status` and `dotf lane plan` from the main checkout, against the herdr already running (never start a herdr server, per #2550 comment 6205806273). Record in `verification.md` that every worktree with a live session is classed `ambiguous`, never `auto`. `apply` is not run live as AC4 evidence: a safety check must not be performed by the reaper it guards against.
- [ ] Verify (same suite). Lesson. Open the PR.

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-127-lane-reconcile/features.json`):

```json
[
  {
    "id": "CLI-127-lane-reconcile-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
