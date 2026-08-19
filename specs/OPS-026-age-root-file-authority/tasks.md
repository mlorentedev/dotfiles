---
tags: [spec, tasks, templates]
created: "2026-08-19"
---

# Tasks - OPS-026-age-root-file-authority

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch from main: `feat/age-root-file-authority`
- [x] `proposal.md` complete; the open question that blocked this for months is
      resolved and recorded — the ADR's ratified example does not validate

## Implementation

- [x] [AC1] `BackendFileAuthority` in `ValidBackends()`, with `checkFileAuthoritySources`
      requiring a file expose with var+path and refusing an age source
- [x] [AC6] `Verifier`, the optional half of `Resolver`, and `fileAuthorityResolver`
      whose `Resolve` REFUSES and whose `VerifyEntry` answers present/mode
- [x] [AC1] `TestParseRegistry_AcceptsExactlyValidBackends` gains a valid fixture per
      backend, and fails when a backend has none. Its single env-shaped template
      reported file-authority rejected while the parser was correct
- [x] [AC2] `AGE_KEY_PERSONAL` in `secrets/registry.yaml`, `bw:` block without a
      folder (the ratified taxonomy has none for floor; placing it is an ADR decision)
- [x] [AC4] 8 unit cases, each observed failing on its own mutation
- [x] [AC7] ADR-028's worked example amended, with all three reasons the old one
      could not validate

## Deferred, and visibly so

- [ ] [AC5] The drift comparison: local fingerprint vs the Bitwarden convenience
      copy, SKIPPED-with-a-reason when there is no session, never OK. Needs a live
      session and `age-keygen -y`; it is the point of **#1000** rather than of this
      spec. `fileAuthorityResolver`'s own comment says the check answers a narrower
      question until this lands, so the gap cannot be mistaken for coverage

## Not this spec's work, found while doing it

- [ ] `dotf spec init` scaffolds `proposal.md`, `tasks.md`, `verification.md` — but
      `contractFiles` requires `features.json` too, so every spec the tool creates is
      born unable to archive. This is why GUARD-003 has no `features.json`. Ticketed;
      not fixed here

## Closing

- [x] Every acceptance criterion has a feature entry with a non-vacuous verification
      command, and the deferred one is marked `deferred` rather than `pending`
- [x] `go build`, `go vet`, `go test ./...` (17 packages), `golangci-lint run` at the
      pinned 2.12.2 — 0 issues
- [x] `dotf secrets verify` — 34 ok, 0 missing, 0 failed (33 before)
- [ ] PR opened referencing this spec folder
- [ ] Adversarial review passes before archive
