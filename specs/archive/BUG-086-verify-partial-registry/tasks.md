---
tags: [spec, tasks, templates]
created: "2026-08-16"
---

# Tasks - BUG-086-verify-partial-registry

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/bug-086-verify-partial-registry`
- [x] `proposal.md` complete, acceptance criteria testable

## Implementation

- [x] [AC6] Split `validate()` into per-secret `validateSecret` + document-level checks
- [x] [AC1] `ParseRegistryPartial` — per-secret validation, defects returned not raised
- [x] [AC6] Reimplement `ParseRegistry` ON TOP of it (one check set, two policies)
- [x] [AC5] Exclude defective secrets from the returned registry
- [x] [AC4] Register id/vars only after a secret validates
- [x] [AC1] `verify` reports each defect as a `FAILED` row
- [x] [AC2][AC3] `scopeVerify` — match args against defects as well as the registry
- [x] Tests, including the mutation check

## Verification

- [x] `go build ./... && go vet ./... && go test ./...` green
- [x] `golangci-lint run` at the pinned 2.12.2 — 0 issues
- [x] Four AC guards observed FAILING with the bug reintroduced
- [x] Real registry unchanged: `33 ok, 0 missing, 0 failed`
