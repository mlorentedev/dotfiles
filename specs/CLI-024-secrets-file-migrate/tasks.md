---
tags: [spec, tasks, templates]
created: "2026-08-14"
---

# Tasks - CLI-024-secrets-file-migrate

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created (stacked on the merged-pending `feat/secrets-bw-migration` tip, per user decision): `feat/secrets-file-migration`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (STRIPE_BACKUP_CODE grouping decided: ship as-declared)

## Implementation

- [x] [P] [AC3] Write failing test `TestSecretsMigrate_FileSecret_ByteExact` (`secrets_migrate_test.go`): a file-secret fixture (`expose: { file: ... }`) with a multi-line value that has NO trailing newline, migrated via `migrateExec`, asserting the bw-written value is byte-identical (no trim, no addition).
- [x] [AC1] [AC3] Relax `assertMigratable` (`registry_write.go`) to permit `s.Expose.File != nil` (skip the multi-var/per-var check, which does not apply to file secrets — `Vars()` already returns exactly one for them); update its doc comment and `SetBackendBW`'s doc comment to describe both shapes.
- [x] [AC4] Fix the stale `#612 M3/M6` comment in `registry_write.go` — that milestone numbering does not exist in #612 (A/B/C phase letters); replace with an accurate description of what's actually still rejected (multi-var/per-var env secrets).
- [x] [AC1] Remove `migrateGuard`'s file-secret rejection block (`secrets_migrate.go`); capture `isFile` from the existing `s.BWTarget("")` third return value (already correctly computed, previously discarded via `_`).
- [x] [AC1] [AC3] Branch `ageValue` on `isFile`: byte-exact (no `TrimRight`) for file secrets, existing trailing-newline trim for env secrets; empty-value guard applies to both.
- [x] [AC1] Thread `isFile` into the `applySet` call and the parity-gate's `normalizeValue(back, isFile)` comparison (both currently hardcode `false`).
- [x] Refactor for clarity; re-run the full Go suite (`go test ./... -count=1`) + `golangci-lint run`.
- [x] [AC1] [AC5] Live: migrate the 5 non-Zoho file secrets (`KUBECONFIG`, `GMAIL_BACKUP_CODE`, `CHATGPT_BACKUP_CODE`, `CHATGPT_RECOVERY_CODE`, `STRIPE_BACKUP_CODE`) against the real Bitwarden vault; SHA-256 spot-check age-side vs. bw-side on at least the multi-line `KUBECONFIG`.
- [x] [AC2] [AC5] Live: confirm `dotf secrets migrate ZOHO_RECOVERY_CODE` still fails with the pre-existing `zoho` item-ambiguity error (#962), not a new error; `dotf secrets verify` reports 33/33 OK overall.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-024-secrets-file-migrate/features.json`):

```json
[
  {
    "id": "CLI-024-secrets-file-migrate-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
