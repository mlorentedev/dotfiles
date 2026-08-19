---
tags: [spec, tasks, templates]
created: "2026-08-19"
---

# Tasks - OPS-032-dr-drift-detection

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/dr-drift-and-manifest`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC1] Add `recipient:` schema parsing and validation on file-authority secrets in `cli/internal/secrets/registry.go`
- [x] [AC2] Add `age-keygen -y` derivation and mismatch comparison in `cli/internal/secrets/fileauthority.go`
- [x] [AC3] Handle cannot-derive errors without swallowing underlying cause; verify registry entries
- [x] [P] [AC4] Implement `EscrowManifest`, `ManifestFrom`, and `AtomicWrite` of `escrow-manifest.json` in `cli/internal/secrets/manifest.go` and `cli/internal/secrets/escrow.go`
- [x] [AC4] Update `sensitive/dr/.gitignore` to un-ignore `!escrow-manifest.json` by exact name
- [x] [P] [AC5] Implement `EscrowManifest.Differs` and wire `checkEscrowDescribesVault` into `cli/internal/doctor/checks_dr.go`
- [x] [AC6] Implement `SKIP` branches for missing manifest and unavailable Bitwarden sessions
- [x] [P] [AC7] Update `docs/runbooks/guide-secrets-governance.md` with offline recovery steps and recipient verification
- [x] [AC8] Verify `go vet` and `GOOS=windows go vet` clean across all CLI packages

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/OPS-032-dr-drift-detection/features.json`):

```json
[
  {
    "id": "OPS-032-dr-drift-detection-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
