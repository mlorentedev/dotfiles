---
tags: [spec, tasks, templates]
created: "2026-08-13"
---

# Tasks - BUG-074-doctor-bw-reach

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `fix/doctor-bw-reach`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — the two
      that mattered (severity source, CI going red) were resolved by inspection
      before implementation, and both are recorded there with their evidence.

## Implementation

- [x] [AC3] Add the `BWBackedSecrets` seam to `System`, reading through the
      checkout-preferring registry path (never `cfg.DotfilesDir`)
- [x] [P] [AC1] Test: `unauthenticated` is reported and names `bw login`
- [x] [AC1] Implement the `bw status` tier
- [x] [P] [AC1] Test: severity follows exposure, asserted in both directions
- [x] [AC1] Implement severity keying off `BWBackedSecrets`
- [x] [P] [AC2] Test: a 45d-old `lastSync` warns while status still reads `locked`
- [x] [AC2] Implement `checkBWSyncAge`, split out so the staleness rule is
      testable independently of vault status
- [x] [P] [AC3] Test: unlocked vault proves reach; sync failure is a FAIL
- [x] [AC3] Implement the `bw sync` round-trip tier
- [x] Refactor: reuse `firstLine` from `checks_vault_hooks.go` rather than
      redeclaring it; wrap it in `bwFailDetail` for the exec-error fallback
- [x] [AC4] Mutation-test all three tiers; observe each guarding test go red

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a
      non-vacuous verification command
- [x] Type checks pass (`go build ./...`, `go vet ./...`)
- [x] Lint passes (`golangci-lint run` → 0 issues, pinned 2.12.2)
- [x] No unrelated changes in the diff (no scope creep) — the diff is the new
      check, its seam, its registration, and its tests
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

See `features.json` alongside this file. States are left at `pending`: per the
gating rule above, an agent may not write `passing` — only the harness may, after
running each `verification` command and capturing exit 0. The evidence for every
criterion (mutation results, live smoke output from both vault states) is in
`verification.md`.
