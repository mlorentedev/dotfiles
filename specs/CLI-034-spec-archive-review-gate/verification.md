---
tags: [spec, verification, templates]
created: "2026-08-09"
---

# Verification - CLI-034-spec-archive-review-gate

## Evidence

Every command below was run in this session on `feat/spec-archive-review-gate`.

- [x] AC1 (blocks without review) -> `TestArchiveBlocksOnMissingReview` — **proved red first** on
      `main` (`review_test.go:20: expected a missing review.md to block the archive`), green after
      the pre-flight landed
- [x] AC2 (blocks on FAIL / unknown / foreign) -> `TestArchiveBlocksOnFailVerdict`,
      `TestArchiveBlocksOnUnknownVerdict`, `TestArchiveBlocksOnForeignReview`
- [x] AC3 (PASS and PASS-WITH-GAPS archive) -> `TestArchiveProceedsOnPass`,
      `TestArchiveProceedsOnPassWithGaps`
- [x] AC4 (declared waiver) -> `TestArchiveWaivedWithReason`,
      `TestArchiveWaivedWithoutReasonRefuses`
- [x] AC5 (`--force-without-review`) -> `TestArchiveForceWithoutReview`,
      `TestArchiveForceWithoutReviewOverridesFail`
- [x] AC6 (staleness) -> `TestGitStalenessDetectsContractChange`,
      `TestGitStalenessUnresolvableShaIsStale`, `TestArchiveBlocksOnStaleReview`
- [x] AC7 (staleness self-defeat guards) -> `TestGitStalenessIgnoresReviewOwnCommit`,
      `TestGitStalenessIgnoresVerificationMd`
- [x] AC8 (skills document the contract) -> the `features.json` grep + `compile-harness.sh --check`
      command, exit 0; render parity verified file-by-file against the vault SSOT

## Test status

- Test suite: `go test ./...` in `cli/` -> all packages ok, no failures
- `go vet ./...` -> clean
- `./scripts/compile-harness.sh --check` -> `no harness drift`
- Manual smoke: `gitStaleness` exercised against real `git init` repositories (not only the fake),
  which is where AC6/AC7 actually live
- No regressions: the eight pre-existing archive tests pass after being updated to the new
  contract (see the first decision below)

## Decisions made during implementation

- **Existing archive tests were updated, not exempted.** Adding a mandatory pre-flight broke eight
  tests that archived specs carrying no `review.md`. That is the contract changing, not the tests
  being wrong: under CLI-034 a minimal archivable spec includes a passing review, so `seedSpec` and
  the affected `writeSpec` maps now say so. Exempting them would have left the suite describing a
  system that no longer exists.
- **Staleness is skipped, not failed, outside a git work tree.** With no history there is nothing
  for the review to be stale against. The checker returns `known=false` and the caller skips —
  presence, verdict and spec-id checks still applied. Failing closed here would break every
  consumer that runs outside a repo without buying any real safety.
- **An unresolvable `reviewed_sha` inside a repo IS stale.** Distinct from the case above: here
  history exists and the sha is not in it (a rebase rewrote it), so the review cannot be shown to
  describe the current change. Refusing is the safe direction and matches the proposal.
- **Spec-id matching was implemented, not just documented.** The frontmatter schema in the proposal
  claimed `spec:` must equal the folder name. Shipping that line without enforcing it would have
  been the exact spec-vs-code mismatch this gate exists to catch, so `TestArchiveBlocksOnForeignReview`
  now covers the copy-paste alibi (a review lifted from a sibling spec).
- **Frontmatter comment-stripping had to respect quotes.** The scaffolded templates carry trailing
  `# comments`, but a quoted value legitimately contains `#` (`issue: "mlorentedev/dotfiles#875"`).
  Naive stripping corrupted the field; `TestFrontmatterKeepsHashInQuotedValue` pins both halves.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? **yes** — a freshness check scoped to the artifact it
      validates is self-defeating; the checked set must exclude the artifact itself and any file
      written at gate time.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? **no** — this implements an existing
      doctrine (determinism by code), it does not choose a new architecture.
- [ ] New pattern candidate for `00_meta/patterns/`? **candidate** — "a gate needs a producer": the
      verifier and the artifact's author are one contract, and specifying either alone ships a
      system that compiles and does not work. Only promote if it recurs in a second project.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-034-spec-archive-review-gate/` -> `specs/archive/…` (via
      `dotf spec archive CLI-034-spec-archive-review-gate --pr <url>`)
- [ ] **`review.md` produced by an INDEPENDENT session** — this spec must pass its own gate, and the
      implementer cannot be the reviewer without voiding the point
- [ ] Bitácora #875 moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed
