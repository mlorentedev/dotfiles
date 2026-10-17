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

- [x] AC4 -> `TestHandoffClaimsBeforeItStarts`, `TestHandoffSkipsToTheNextFreeItemOnALoss`, `TestHandoffStartsNothingWhenEveryItemIsClaimed`, `TestHandoffFailsClosedWhenTheClaimCannotBeTaken`, `TestHandoffDoesNotClaimAWorktreeThatExists`, `TestHandoffDryRunTakesNoClaimButSkipsAHeldItem`, `TestHandoffReleasesTheClaimWhenTheLaneDoesNotStart` (review finding: a start that creates no worktree must not leak the claim), `TestHandoffKeepsTheClaimOnceTheWorktreeExists` (CodeRabbit: after `Create` the next run skips the claim, so it must stay), `TestClaimIgnoresARefThatOnlyEndsLikeTheClaim` (`ls-remote` matches by tail)
- [x] AC5 -> `TestLaneRelease` (real git, two checkouts), primitive `TestClaimRelease`

### PR 3 (the decision on record)

- [x] AC7 -> `TestClaimRefFormOnARemote` passed on GitHub (`mlorentedev/dotfiles`, 6.9s) and on Gitea (`gitea.kubelab.live/personal/imagesensortool`, 14.9s), each followed by an empty `ls-remote 'refs/dotf/*'`; `gh run list --event push` shows no workflow for the ref. ADR-050 decision 2 amended from open to decided.
- [x] `lane claims` shows every claim with its holder and age -> `TestClaimList`, `TestClaimRowsFlagsAClosedIssueAndLeavesAnUnreadableOneOpen`, `TestLaneClaimsListsWhatTheRemoteHolds`

### Review round 1 (FAIL, fixed in the following round)

- Major, REAL: the one production construction of `Options.Claims` (`doLaneNext`) was untested -> applied: `TestLaneNext_DryRunNamesTheNextIssueAndChangesNothing` asserts the `ls-remote origin refs/dotf/claims/5` probe is asked; deleting `Claims:` from `doLaneNext` makes it fail (mutation checked).
- Minor, REAL: checkbox drift in `tasks.md` and `proposal.md` -> applied. AC7 contradiction in this file -> applied (stale paragraph removed).
- Minor, REAL: `handoffIssue` complexity 18 and `List` 11 -> applied: `findLane`, `startLane` and `claimIssues` extracted; both are under 10 (measured with gocyclo at min-complexity 10).
- Minor, UNTESTED: the claim push skips the pre-push hook -> applied: `TestClaimPushSkipsThePrePushHook` (a failing hook refuses a plain push and not the claim; removing `--no-verify` fails it).
- Minor, SPECULATIVE: two holders whose paths differ only in a control character collide -> declined: a claim is coordination, not an authorisation boundary, and such a path is not one a worktree root produces.
- Minor, THEORETICAL: `List` aborts if a claim vanishes between `ls-remote` and `fetch` -> declined: the listing is a diagnostic and a second run answers; swallowing the error would hide a real read failure.
- Disclosure, not a defect: the review range contains three unrelated merged PRs (#2622, #2623, #2582) because they interleaved on main.

## Test status

- Test suite: `cd cli && go test ./internal/lane/ -run TestClaim -count=20` -> ok; `go vet`, `GOOS=windows go vet`, `golangci-lint run ./internal/lane/...` -> 0 issues
- Manual smoke test: `TestClaimRefFormOnARemote` against a local bare remote -> PASS; GitHub and Gitea not yet probed (see above)
- No regressions in existing test suite: yes (PR 1 adds files only)

## Decisions made during implementation

- Every claim commit carries a random nonce: an identical commit pushed over itself is "up to date" and would let two same-holder claims both win (lesson 459).
- The claim time is read from the commit message, not the commit date, so a test (and a future caller) controls it; an unreadable time prints as unknown rather than blocking a release.
- The holder is the checkout (`user@host:<worktree>`), not the process: a per-process nonce would make every `lane release` from a later invocation another holder's and force `--force`; a per-user id would make two same-user sessions one claimant.
- `lane next` does not set the forge assignee: `Next` skips assigned issues, so a second invocation for the same item would start a second lane. A claim whose worktree already exists is not re-claimed (a second run is not a loss); a claim held by the same checkout from a run that died before creating the worktree fails closed and says `lane release`.
- The probe ran against live repositories rather than a scratch one (the ADR said scratch): the ref number is one no queue holds, the test deletes it, and `ls-remote` was empty afterwards on both. The owner authorised the push to GitHub and named the Gitea host; the Gitea repository (`personal/imagesensortool`) was the one already configured on the machine.
- Release on land is not automatic. A claim's job is to decide who starts an item; `Next` skips closed issues, so a claim left on a landed one blocks nothing. `lane claims` flags it and `release --force` clears it. An automatic release would need the sweeper (another checkout than the holder) to force-release, a second authority over the claim: left until the ref litter matters.
- The empty tree is the constant SHA-1 id: SHA-256 repositories are out of scope until one exists in the fleet.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-459-pushing-an-object-the-ref-already-points-at-is-not-a-refusal.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-050-one-lane-contract-for-every-repo-agent-and-forge.md
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: single-repo mechanism, no second project uses it yet

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-128-lane-claim-ref/` -> `specs/archive/CLI-128-lane-claim-ref/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
