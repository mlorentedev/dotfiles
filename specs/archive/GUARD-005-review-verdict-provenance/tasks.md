---
tags: [spec, tasks, templates]
created: "2026-08-21"
---

# Tasks - GUARD-005-review-verdict-provenance

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/review-verdict-provenance`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left

## Implementation

- [x] `ReviewRequest` sidecar type + `WriteReviewRequest` / `ReadReviewRequest`
- [x] `fileDigest` returning "" for unreadable, so the comparison fails OPEN into "changed"
- [x] `checkReviewProvenance` wired into the archive gate BEFORE the verdict and staleness checks
- [x] Digest check ordered first, so a no-verdict run is not reported as a sha mismatch
- [x] Failing tests for all three refusals, plus the silent-without-sidecar and loud-on-damage cases
- [x] **Launcher wiring** — `dotf spec review` writes the sidecar before the runner starts
- [x] Write failure warns rather than blocking the launch
- [x] End-to-end reproduction against the real binary

## Closing

- [x] Every acceptance criterion covered by at least one test
- [x] `features.json` verifiers propagate the runner exit status and pin tests by unique name
- [x] `go build` / `go vet` / `go test ./...` green
- [x] Proof artifacts removed; the working tree carries only the change
- [x] `verification.md` filled in with this session output
- [x] PR opened referencing this spec folder
