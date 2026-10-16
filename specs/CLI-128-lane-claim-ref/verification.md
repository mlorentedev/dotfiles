---
tags: [spec, verification, templates]
created: "2026-10-16"
---

# Verification - CLI-128-lane-claim-ref

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

### PR 1 (the claim primitive)

- [x] AC1 -> `TestClaimAcquire` (same and other holder refused; found while writing it: an identical claim commit is "up to date", not a refusal, so every claim carries a nonce, lesson 459)
- [x] AC2 -> `TestClaimedErrorNamesTheHolder`
- [x] AC3 -> `TestClaimConcurrent` (8 claimants, 1 winner, 20 runs green)
- [x] AC6 -> `TestClaimIsInvisibleToADefaultClone`
### PR 2 (wiring)

- [x] AC4 -> `TestHandoffClaimsBeforeItStarts`, `TestHandoffSkipsToTheNextFreeItemOnALoss`, `TestHandoffStartsNothingWhenEveryItemIsClaimed`, `TestHandoffFailsClosedWhenTheClaimCannotBeTaken`, `TestHandoffDoesNotClaimAWorktreeThatExists`, `TestHandoffDryRunTakesNoClaimButSkipsAHeldItem`, `TestHandoffReleasesTheClaimWhenTheLaneDoesNotStart` (review finding: a failed start must not leak the claim), `TestClaimIgnoresARefThatOnlyEndsLikeTheClaim` (`ls-remote` matches by tail)
- [x] AC5 -> `TestLaneRelease` (real git, two checkouts), primitive `TestClaimRelease`
- [ ] AC7 (forge probe) -> PR 3

The forge probe is `TestClaimRefFormOnARemote`, opt-in through `DOTF_CLAIM_PROBE_REMOTE`; it passes against a local bare remote. A push of a probe ref to the real `origin` was refused by the harness, so GitHub and Gitea are still unprobed and AC7 stays open until the owner runs it.

## Test status

- Test suite: `cd cli && go test ./internal/lane/ -run TestClaim -count=20` -> ok; `go vet`, `GOOS=windows go vet`, `golangci-lint run ./internal/lane/...` -> 0 issues
- Manual smoke test: `TestClaimRefFormOnARemote` against a local bare remote -> PASS; GitHub and Gitea not yet probed (see above)
- No regressions in existing test suite: yes (PR 1 adds files only)

## Decisions made during implementation

- Every claim commit carries a random nonce: an identical commit pushed over itself is "up to date" and would let two same-holder claims both win (lesson 459).
- The claim time is read from the commit message, not the commit date, so a test (and a future caller) controls it; an unreadable time prints as unknown rather than blocking a release.
- The holder is the checkout (`user@host:<worktree>`), not the process: a per-process nonce would make every `lane release` from a later invocation another holder's and force `--force`; a per-user id would make two same-user sessions one claimant.
- `lane next` does not set the forge assignee: `Next` skips assigned issues, so a second invocation for the same item would start a second lane. A claim whose worktree already exists is not re-claimed (a second run is not a loss); a claim held by the same checkout from a run that died before creating the worktree fails closed and says `lane release`.
- The empty tree is the constant SHA-1 id: SHA-256 repositories are out of scope until one exists in the fleet.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-459-pushing-an-object-the-ref-already-points-at-is-not-a-refusal.md
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: pending PR 3, which amends ADR-050 decision 2 once the forge probe has run
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: single-repo mechanism, no second project uses it yet

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-128-lane-claim-ref/` -> `specs/archive/CLI-128-lane-claim-ref/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
