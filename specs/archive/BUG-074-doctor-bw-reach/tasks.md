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

## Round 3 — second review remediation

Review 2 returned **FAIL** (0 Blockers, 3 Majors, all REAL). Every finding was
reproduced independently before being accepted; none was argued down.

- [x] [AC1-AC3] Split the streams. `CommandOutputBounded` returned
      `CombinedOutput`, so one line of `bw` stderr chatter broke the
      `json.Unmarshal` and the check returned early with **all three tiers
      skipped** — on `bw`'s first invocation on a machine, i.e. exactly the fresh
      box `setup-linux.sh` has just provisioned. Reproduced live before fixing.
      The seam now returns `(stdout, stderr, err)`; `bw status` parses stdout.
- [x] [AC1] Key the tier-3 sync failure to exposure, like the `unauthenticated`
      branch already was. A flat `rep.Fail` exited doctor 1 for an offline
      machine at zero exposure, contradicting `## What`, the check's own header
      comment, and doctor's precedent for an unreachable remote
      (`checks_pat.go`).
- [x] [AC4] Prove the check is **registered**: deleting `checkBitwardenReach`
      from `Run()` left all 13 packages green, since every test called it
      directly. `TestRun_RegistersTheBitwardenReachSection` closes that, mirroring
      the existing `TestRun_QuickSkipsHeavySections` assertion pattern.
- [x] Minor: `rep.Fix` was emitted on a read-only run, making the summary report
      `Applied 1 fix action(s)` for a repair nothing performed. The recovery
      command moved into the FAIL message.
- [x] Minor: `bwFailDetail` now prefers stderr, where `bw` actually writes its
      errors — the merged read surfaced startup chatter as if it were the cause.
- [x] Minor: document the third CI trigger in `proposal.md` risk 3 — `bw` is an
      npm-sourced tool in `packages.json`, so adding node/npm to the integration
      image for any unrelated reason installs it and activates this check.
- [x] Minor: document the `RepoDir` cwd walk-up, which degrades severity when
      doctor runs from another git repo.
- [x] **Question answered**: the 30d threshold is an educated floor, not a
      derived one. The incident bounds the token dead *by* 45d; it does not show
      it alive at 30d, and no upstream lifetime is cited. `proposal.md` now says
      "earlier than the only expiry we have observed" instead of "while the token
      is still renewable".
- [x] Mutation battery re-run: 5 mutants, **5 detected**, each with a guard that
      aborts when the mutation fails to apply — the first attempt reported two
      false SURVIVEDs because the `sed` pattern silently matched nothing.
- [ ] Fresh adversarial review on a **non-Anthropic** model (NaN primary). Round 2
      was Anthropic-on-Anthropic; the standing rule now forbids that.

## Machine-readable features

See `features.json` alongside this file. States are left at `pending`: per the
gating rule above, an agent may not write `passing` — only the harness may, after
running each `verification` command and capturing exit 0. The evidence for every
criterion (mutation results, live smoke output from both vault states) is in
`verification.md`.
