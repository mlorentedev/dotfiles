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

- [ ] Branch created from main: `feat/lane-claim-ref`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" that block the first PR (the forge probe is task-tracked below)

## Implementation

### PR 1: the claim primitive

- [ ] [AC1] Write `cli/internal/lane/claim_test.go` (`TestClaimAcquire`) over a real local bare remote: the first Acquire wins, the second with the same sha and with another sha is refused. Expected: FAIL.
- [ ] [AC2] Add `TestClaimedErrorNamesTheHolder`: the loser's `ClaimedError` carries the holder and the claim time read from the claim commit.
- [ ] [AC3] Add `TestClaimConcurrent`: N goroutines Acquire one issue against one bare remote; exactly one wins and the rest hold a `ClaimedError`.
- [ ] [AC6] Add `TestClaimIsInvisibleToADefaultClone`: after Acquire, a default clone's `fetch` and its branch list carry no `refs/dotf` entry.
- [ ] [AC1] Write `cli/internal/lane/claim.go` until they pass: `Git` seam, `Acquire`, `Holder`, `Release`, `ClaimedError`; the claim commit built with `git commit-tree` on the empty tree; the push skips hooks.
- [ ] [AC7] Add the opt-in `TestClaimRefFormOnARemote` (skipped unless `DOTF_CLAIM_PROBE_REMOTE` names a remote): win, refusal, delete. Run it against GitHub with the owner's go-ahead and record the output in `verification.md`.
- [ ] Verify (shell and Go loops), lesson, open the PR.

### PR 2: `lane next`, `lane claim`, `lane release`

- [x] [AC4] `TestHandoffClaimsBeforeItStarts`, `TestHandoffSkipsToTheNextFreeItemOnALoss`, `TestHandoffStartsNothingWhenEveryItemIsClaimed`, `TestHandoffFailsClosedWhenTheClaimCannotBeTaken`, `TestHandoffDoesNotClaimAWorktreeThatExists`, `TestHandoffDryRunTakesNoClaimButSkipsAHeldItem` (all failed before the wiring; a mutation that disables the claim fails 5)
- [x] [AC4] `Handoff` claims through `Options.Claims` before `launch`
- [x] [AC5] `lane claim` and `lane release [--force]` and `TestLaneRelease` (real git, two checkouts)
- [x] Verify (build, vet, `GOOS=windows go vet`, `go test ./...`, golangci-lint), open the PR.

### PR 3: the decision on record

- [ ] `lane status` shows a claim and its age (read-only, no new store).
- [ ] [AC7] Amend ADR-050 decision 2 from "open" to the decided form, naming the forges the probe passed on; record any forge that refused.
- [ ] Close #2551 with the evidence; answer the promotion candidates in `verification.md`.

## Machine-readable features

`features.json` carries one entry per behavior with the command that proves it. State is harness-owned.
