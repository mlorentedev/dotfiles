---
tags: [spec, tasks, templates]
created: "2026-10-16"
---

# Tasks - CLI-128-lane-claim-ref

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/lane-claim-ref`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" that block the first PR (the forge probe is task-tracked below)

## Implementation

### PR 1: the claim primitive

- [x] [AC1] Write `cli/internal/lane/claim_test.go` (`TestClaimAcquire`) over a real local bare remote: the first Acquire wins, the second with the same sha and with another sha is refused. Expected: FAIL.
- [x] [AC2] Add `TestClaimedErrorNamesTheHolder`: the loser's `ClaimedError` carries the holder and the claim time read from the claim commit.
- [x] [AC3] Add `TestClaimConcurrent`: N goroutines Acquire one issue against one bare remote; exactly one wins and the rest hold a `ClaimedError`.
- [x] [AC6] Add `TestClaimIsInvisibleToADefaultClone`: after Acquire, a default clone's `fetch` and its branch list carry no `refs/dotf` entry.
- [x] [AC1] Write `cli/internal/lane/claim.go` until they pass: `Git` seam, `Acquire`, `Holder`, `Release`, `ClaimedError`; the claim commit built with `git commit-tree` on the empty tree; the push skips hooks.
- [x] [AC7] Add the opt-in `TestClaimRefFormOnARemote` (skipped unless `DOTF_CLAIM_PROBE_REMOTE` names a remote): win, refusal, delete. Run it against GitHub with the owner's go-ahead and record the output in `verification.md`.
- [x] Verify (shell and Go loops), lesson, open the PR.

### PR 2: `lane next`, `lane claim`, `lane release`

- [x] [AC4] `TestHandoffClaimsBeforeItStarts`, `TestHandoffSkipsToTheNextFreeItemOnALoss`, `TestHandoffStartsNothingWhenEveryItemIsClaimed`, `TestHandoffFailsClosedWhenTheClaimCannotBeTaken`, `TestHandoffDoesNotClaimAWorktreeThatExists`, `TestHandoffDryRunTakesNoClaimButSkipsAHeldItem` (all failed before the wiring; a mutation that disables the claim fails 5)
- [x] [AC4] `Handoff` claims through `Options.Claims` before `launch`
- [x] [AC5] `lane claim` and `lane release [--force]` and `TestLaneRelease` (real git, two checkouts)
- [x] Verify (build, vet, `GOOS=windows go vet`, `go test ./...`, golangci-lint), open the PR.

### PR 3: the decision on record

- [x] `lane claims [--json]` lists every claim with holder, age and an "issue closed, releasable" note (replaces "lane status shows a claim": status is one row per worktree and its JSON is a bare array, a claim has no worktree)
- [x] [AC7] Probe GitHub and Gitea with `TestClaimRefFormOnARemote`; amend ADR-050 decision 2 from "open" to decided, naming both forges.
- [x] Release on land or park is deliberately not automatic (see `verification.md`, decisions).
- [x] Round 1 of `dotf spec review` (nan/mimo-v2.6-flash): FAIL, one Major (the `lane next` wiring was mutation-undetected, lesson 457's class). Fixed: `TestLaneNext_DryRunNamesTheNextIssueAndChangesNothing` now asserts the claim probe is asked, and deleting `Claims:` from `doLaneNext` fails it; plus `TestClaimPushSkipsThePrePushHook`, `handoffIssue` and `List` split under complexity 10.
- [ ] Round 2 of `dotf spec review CLI-128-lane-claim-ref` by a non-implementer, archive on PASS, close #2551 with the evidence.

## Machine-readable features

`features.json` carries one entry per behavior with the command that proves it. State is harness-owned.
