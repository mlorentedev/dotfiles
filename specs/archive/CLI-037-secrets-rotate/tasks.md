---
tags: [spec, tasks, templates]
created: "2026-08-15"
---

# Tasks - CLI-037-secrets-rotate

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Command shipped before the spec was filled: #1003 (`36448473`), then the write path on `bw serve` in #1007 (`fe2f1913`)
- [x] `proposal.md` filled from #996 (2026-10-10)

## Implementation

- [x] [AC1] [AC2] [AC3] Tests for write + sync + read-back, the unchanged-fingerprint failure, the no-op refusal and the refusal to create; `secrets_rotate.go` (#1003)
- [x] [AC5] `--dry-run` test and implementation (#1003)
- [x] [AC6] Write path through `bw serve` (#1007)
- [x] [AC4] Failing tests for the probe: a live token passes and is reported, a refused one fails the rotation, and the probe receives the new value. The code was in #1003; the tests were missing. Two mutations (drop the probe call, swallow its error) each fail them
- [x] [AC7] Failing tests for `--push-ci`: both CI consumers get the new value and nothing else, no flag uploads nothing, no CI consumer is a note, `--dry-run` names the repos
- [x] [AC7] Extract `pushCI` from `secrets sync ci` and implement `--push-ci` on it, so a rotated value reaches CI through the same skip rules. Four mutations each killed

## Closing

- [x] Every acceptance criterion is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `go build`, `go vet` (also `GOOS=windows`), `go test ./...` and golangci-lint pass
- [x] `verification.md` filled in
- [ ] Independent adversarial review, then archive
