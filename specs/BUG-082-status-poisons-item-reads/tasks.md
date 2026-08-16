---
tags: [spec, tasks, templates]
created: "2026-08-16"
---

# Tasks - BUG-082-status-poisons-item-reads

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/bug-082-status-poisons-item-reads-v2`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] Open questions resolved or mitigated structurally (see Risks)

## Implementation

- [x] Characterise the trigger: probe → 10 item GETs, per candidate endpoint
- [x] Falsify the standing hypotheses (reuse, concurrency, compression, spacing)
- [x] [AC3] Add `probeReadable`, replacing the `/status` probe in `SelectBWBackend`
- [x] [AC4] Teach the fake that a locked daemon refuses data endpoints
- [x] [AC3] Guard: `TestSelectBWBackend_NeverCallsStatus`, all three daemon states
- [x] [AC1][AC2] Verify live against the real daemon
- [x] Report the trigger upstream

## Verification

- [x] `go build ./... && go vet ./... && go test ./...` green
- [x] `golangci-lint run` at the pinned 2.12.2 — 0 issues
- [x] AC1: 10/10 `secrets run` (was 2/10)
- [x] AC2: 6 × `secrets verify` = 198 resolutions, 0 failures
