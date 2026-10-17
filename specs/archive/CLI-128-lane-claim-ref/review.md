---
spec: "CLI-128-lane-claim-ref"
verdict: "PASS"
reviewed_sha: "d3f0038565b16241a1881e53847a5e431696984a"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-10-16"
---

## Adversarial review

**Scope**: CLI-128-lane-claim-ref — `git diff 6f7aba20bc75959fe58e843915a63ea61babbfc9...HEAD` (base resolved by the launcher), i.e. the four spec commits (#2624, #2625, #2627, #2636) plus interleaved commits from main (#2622, #2623, #2582, #2630, #2628, #2629, #2632).

**Sources**: `specs/CLI-128-lane-claim-ref/{proposal,tasks,verification}.md`, `features.json`, `review-request.json`; `cli/internal/lane/{claim,claim_test,handoff,handoff_claim_test,queue}.go`; `cli/internal/cmd/{lane,lane_test,lane_claim,lane_claim_test,lane_claims}.go`; `docs/adr/adr-050-one-lane-contract-for-every-repo-agent-and-forge.md`; `docs/lessons/{lesson-459-*,lesson-460-*}`.

### Spec and task alignment

- **Contract integrity**: Recomputed normalized digests of `proposal.md`, `tasks.md`, and `features.json` match `review-request.json` byte-for-byte; verified via `go test ./internal/spec/ -run TestReviewRequestRecordsContractDigests`.
- **Round 1 F1 resolution (wiring guard)**: The sole production construction of `Options.Claims` in `doLaneNext` (`cli/internal/cmd/lane.go:147`) is now guarded. Tested in this session:
  - Mutation check: Removing `Claims:` from `doLaneNext` causes `TestLaneNext_DryRunNamesTheNextIssueAndChangesNothing` and `TestLaneNext_ARepositoryUnderAPathWithASpaceHandsTheLaneOver` to fail immediately (`lane next never asked the remote about the claim on #5`). Restoring the line restores green tests.
- **Round 1 F5 resolution (hook-skip test)**: `TestClaimPushSkipsThePrePushHook` now verifies `--no-verify`. Tested in this session:
  - Mutation check: Removing `--no-verify` from `c.git("push", ...)` in `cli/internal/lane/claim.go:101` causes `TestClaimPushSkipsThePrePushHook` to fail immediately (`pre-push ran`). Restoring `--no-verify` restores green.
- **Round 1 F4 resolution (complexity split)**: `handoffIssue` in `cli/internal/lane/handoff.go` was split into `findLane`, `startLane`, and `claim` (reduced from 57 physical lines and CC 18 to 33 lines and CC <= 10). `Claimer.List` was split with `claimIssues` extracted (reduced from CC 11 to CC <= 5 / 24 lines). Verified via `golangci-lint` with `gocyclo` at min-complexity 10: 0 new production functions exceed CC 10 or 40 lines.
- **Round 1 F2 resolution (checkbox drift)**: In `proposal.md`, AC1–AC7 are all marked `[x]`. In `tasks.md`, all implementation tasks across PR 1, PR 2, and PR 3 are marked `[x]`. Only the current round 2 review task remains open.
- **AC1–AC3, AC5, AC6**: Each has a named real-git test. All four `features.json` verification commands executed and passed in this session:
  - `CLI-128-lane-claim-ref-f1`: `TestClaimAcquire`, `TestClaimedErrorNamesTheHolder` -> PASS.
  - `CLI-128-lane-claim-ref-f2`: `TestClaimConcurrent` (8 claimants, 20 runs), `TestClaimIsInvisibleToADefaultClone` -> PASS.
  - `CLI-128-lane-claim-ref-f3`: `TestHandoffClaimsBeforeItStarts`, `TestHandoffSkipsToTheNextFreeItemOnALoss` -> PASS.
  - `CLI-128-lane-claim-ref-f4`: `TestLaneRelease` -> PASS.
- **AC4**: Proven by nine named tests in `internal/lane/` (`TestHandoffClaimsBeforeItStarts`, `TestHandoffSkipsToTheNextFreeItemOnALoss`, `TestHandoffStartsNothingWhenEveryItemIsClaimed`, `TestHandoffFailsClosedWhenTheClaimCannotBeTaken`, `TestHandoffDoesNotClaimAWorktreeThatExists`, `TestHandoffDryRunTakesNoClaimButSkipsAHeldItem`, `TestHandoffReleasesTheClaimWhenTheLaneDoesNotStart`, `TestHandoffKeepsTheClaimOnceTheWorktreeExists`, `TestClaimIgnoresARefThatOnlyEndsLikeTheClaim`), plus command-level tests in `internal/cmd/`.
- **AC7**: Opt-in test `TestClaimRefFormOnARemote` verified against local bare remote. Live probe execution on GitHub and Gitea recorded in `verification.md` and ADR-050 Decision 2; `git ls-remote origin 'refs/dotf/*'` confirmed empty on live origin.
- **Verification evidence executed in this session**:
  - `cd cli && go test ./...` -> PASS (all packages).
  - `cd cli && go vet ./...` -> 0 issues.
  - `cd cli && GOOS=windows go vet ./internal/lane/... ./internal/cmd/...` -> 0 issues.
  - `cd cli && golangci-lint run ./internal/lane/... ./internal/cmd/...` -> 0 issues.
  - `bats tests/roster-consistency.bats tests/herdr-config.bats tests/skills-pipeline.bats tests/compile-harness.bats tests/verify-setup.bats` -> 196 tests, 0 failures, 73 skipped (integration-only).
- **Draft tags**: None found in `specs/CLI-128-lane-claim-ref/`.
- **Promotion candidates**: Answered completely in `verification.md` (Lesson 459 promoted; ADR-050 amended; Pattern candidate declined with reason).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | spec / verification | `verification.md` line 42 contains a lingering stale sentence ("GitHub and Gitea not yet probed (see above)") that contradicts line 26 ("TestClaimRefFormOnARemote passed on GitHub... and on Gitea..."), line 51, and ADR-050 Decision 2. Round 1 removed the stale closing paragraph, but this line was left in the "Test status" section. | Quoted verbatim from `verification.md:42`; `verification.md:26` records the probe times (6.9s / 14.9s) and ADR-050 records it decided. | n/a (documentation inconsistency) | verification.md (outside contract set; free to update without re-review) |
| Minor | THEORETICAL | resilience | `Claimer.List` aborts the entire `lane claims` output if a claim ref vanishes on the remote between `ls-remote` and `fetch`, or if reading a seen commit fails. As noted in Round 1, `lane claims` is a manual diagnostic command and re-running succeeds once the ref settles. | Code inspection of `List()` in `cli/internal/lane/claim.go:200-208` | `TestClaimList` (happy path) | code (surface only; not gating) |
| Minor | SPECULATIVE | identity | `who()` maps control characters to `?`, so two worktrees whose paths differ only in control characters would collide as one holder. In practice worktree paths do not contain control characters. | `cli/internal/lane/claim.go:76` | `TestClaimHolderWithControlCharactersRoundTrips` | code (surface only; not gating) |
| — | — | scope (disclosure) | The review scope `6f7aba20bc75959fe58e843915a63ea61babbfc9...HEAD` contains seven interleaved merged commits from main (#2622, #2623, #2582, #2630, #2628, #2629, #2632) in addition to the spec's four commits (#2624, #2625, #2627, #2636). All commits in the range were tested by the full Go suite, vet, and bats, and pass cleanly. | `git log --oneline` of review scope; test suite exit 0 | n/a | none (disclosed, not a defect) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All acceptance criteria AC1–AC7 verified; command-level dry-run now proves the production claim probe; extensive negative paths covered. |
| Verification       | A | Reproducible commands for all 4 features in features.json pass; mutation checks confirmed tests turn red when guards are removed. |
| Scope              | B | The spec's four commits adhere strictly to proposal scope; interleaved commits from main in the launcher's range are tested and disclosed. |
| Reliability        | B | Atomic CAS via empty expected value with force-with-lease, fail-closed handling, nonce uniqueness, and automatic release on start failure. |
| Maintainability    | A | All new/refactored production functions have CC <= 10 and length <= 40 lines (handoffIssue CC <= 10 / 33 lines, List CC <= 5 / 24 lines). |
| Handoff-readiness  | B | ADR-050 amended to decided, lessons 459 and 460 created and indexed, promotion questions answered; minor residual line in verification.md. |

### Verdict
PASS

### Recommended next steps

1. **verification.md (free — outside the contract set)**: Clean up line 42 of `verification.md` to remove the stale clause `"GitHub and Gitea not yet probed (see above)"`, aligning the section with line 26 and the ADR.
2. **spec lifecycle**: Mark `specs/CLI-128-lane-claim-ref/proposal.md` status as `archived` and proceed with `dotf spec archive CLI-128-lane-claim-ref`.

**(a) Verdict**: PASS.
**(b) `dotf spec archive` / `/spec archive` is advisable**: Yes, the gate requirements are satisfied (fresh passing review, valid frontmatter matching reviewer pool, contract digests verified, promotion candidates answered, no draft tags).
**(c) Actions**: Proceed with archive after addressing the minor `verification.md` note if desired.
