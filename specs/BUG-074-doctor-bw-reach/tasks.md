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
- [x] PR opened referencing this spec folder (#950)

## Round 2 — adversarial review remediation

Review 1 returned **FAIL** (0 Blockers, 3 Majors). Every finding was reproduced
independently before being accepted; none was argued down.

- [x] [AC1] Test the severity **producer**, not just the seam — `bwBackedSecrets`
      against a temp registry (`TestBWBackedSecrets_CountsOnlyBWBackend`,
      `_ZeroWhenNothingMigrated`, `_MissingRegistryErrors`). Kills the reviewer's
      surviving mutant.
- [x] Fix the defect that writing those tests exposed: the counter read
      `env.ResolveRegistryPath`, which silently falls back to the deployed copy
      when the checkout registry is missing. Now `env.RepoRegistryPath`.
- [x] Bound the network subprocesses — `CommandOutputBounded` seam, applied to
      `bw status` (15s) and `bw sync` (45s), tested against the production
      closure (`TestCommandOutputBounded_KillsAnOverrunningCommand`).
- [x] Rewrite `proposal.md` risk 3 with the real chain: doctor **does** run in CI
      via `Dockerfile.integration` → `setup-linux.sh:1505`, and the conclusion
      rests on two named safeguards.
- [x] Minor: retire the overclaiming `bw (... live secrets SSOT) found` PASS line
      in `checks_secrets_tooling.go` — it now says only what PATH presence proves.
- [x] Minor: guard negative sync age (clock skew), with its own mutation-verified
      test.
- [x] Correct the `verification.md` mutation table, which overstated round 1.
- [x] Answer the review's open **Question** instead of leaving it hanging: the
      "a periodic `dotf doctor` is the keep-alive" justification does not hold —
      tier 3 needs an unlocked vault, the resting state is locked, and nothing
      schedules doctor. Claim withdrawn in the code comment and in `proposal.md`;
      the prevention claim reassigned to tier 2, which needs no session.
- [x] Execute the promotion `verification.md` already declared: the
      `docs/lessons.md` entry *a health check that reads local state proves the
      liveness of nothing*. Its index was 3 entries behind the body, so it was
      regenerated rather than appended to.
- [x] Fresh adversarial review **requested** against this head. Recorded before
      the review runs, not after: `tasks.md` is a contract file
      (`cli/internal/spec/review.go:23`), so ticking this box afterwards would
      stale the very verdict the box refers to. No verdict is claimed here — it
      lives in `review.md`, which is excluded from the contract set for exactly
      this reason.

## Machine-readable features

See `features.json` alongside this file. States are left at `pending`: per the
gating rule above, an agent may not write `passing` — only the harness may, after
running each `verification` command and capturing exit 0. The evidence for every
criterion (mutation results, live smoke output from both vault states) is in
`verification.md`.
