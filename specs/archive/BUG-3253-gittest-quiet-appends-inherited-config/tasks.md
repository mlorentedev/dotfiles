---
tags: [spec, tasks, templates]
created: "2026-11-07"
id: "BUG-3253-gittest-quiet-appends-inherited-config"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-3253-gittest-quiet-appends-inherited-config

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`.

## Setup

- [x] Branch created from main: `feat/gittest-env-append`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC4] `TestQuietKeepsTheCallersConfig` — replaces the panic-on-foreign-count test; fails while `Quiet` still refuses.
- [x] [AC1] [AC2] `TestQuietAppendsAfterAForeignEnvironment` — the exact harness shape: three pinned pairs, none of them ours.
- [x] [AC1] [AC2] Implement: `quietPairs` as ordered pairs; `Quiet` reads the caller's pairs and appends the missing ones at `len(caller)+added`, then sets `GIT_CONFIG_COUNT`; `callerPairs()` and `setPair()` added, `alreadyQuiet()` deleted.
- [x] [AC3] Rewrite `TestQuietAcceptsItsOwnKeys` as idempotence: a second `Quiet` adds nothing.
- [x] [AC4] `TestQuietRefusesAMalformedCount` — `abc` and `-1` still panic, since git reads the same number.
- [x] `cleanEnv` clears every `GIT_CONFIG_*` by prefix, so the suite is independent of whatever harness it runs under.
- [x] A/B proof: with `GIT_CONFIG_COUNT=3` intact, stashing the change panics `internal/cmd` at `gittest.go:50`; restoring it exits 0.
- [x] Update `docs/lessons/lesson-517-...md` Solution paragraph (panic -> append, cites #3253).

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass (`go vet ./internal/gittest`)
- [x] Lint passes (`gofmt -l` clean on Linux CI; CRLF makes the local signal unusable on Windows)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [x] Adversarial review round 1 dispositions applied (`review.md`, verdict FAIL: one REAL Major + two code Minors)
- [x] `TestQuietOutranksAConflictingCallerPair` — the Major: a caller pair on a key Quiet also sets, asserting Quiet's value is what a child reads
- [x] `TestQuietRefusesBeforeItTouchesTheEnvironment` — the two Minors: refusal ordered before `quietSystem` writes anything, and the count read pair-by-pair so a huge one costs nothing
- [x] `TestQuietOutranksACallerThatSetsTheSameKeyTwice` — PR round 2 (CodeRabbit Minor): the dedupe tests the caller's *last* pair for the key, not "the pair appears anywhere", so a caller that pins the key twice is still outranked. A/B: reverted to `slices.Contains`, it reports `gc.auto = "1", want "0"`; restored, it passes.
- [x] Rescope AC1 to startup-only — PR round 2 (CodeRabbit Major): `-run '^$'` cannot evidence a full-suite claim, so the claim now matches the predicate and CI owns the full run
- [x] Refusal test asserts the panic names `GIT_CONFIG_COUNT`, so it cannot pass on an unrelated panic — PR round 2 (pr-agent REAL)
- [ ] Adversarial review round 2 (a REAL Major requires re-review; round 1 is superseded)
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` following [[pattern-feature-list-as-primitive]].

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state.
