---
tags: [spec, verification, templates]
created: "2026-11-09"
id: "HARNESS-158-review-scope"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-158-review-scope

## Evidence

- [x] AC1 -> `TestOwnCommitsHoldsOnlyTheSlicesOfTheSpec` (fixture: two slices, unrelated work between and after, a side branch merged `--no-ff`)
- [x] AC2 -> `TestOwnCommitsHoldsOnlyTheSlicesOfTheSpec` (issue-only slice in, `#17270` decoy and the merge naming the issue out), `TestOwnCommitsWithoutAnIssueUsesTheFolderAlone`
- [x] AC3 -> `TestOwnCommitsFollowsARenamedSpecFolder`
- [x] AC4 -> `TestReviewPromptNamesTheOwnCommits`, `TestReviewPromptSaysTheFolderAloneDecidedWithoutAnIssue`, `TestInitialReviewPromptCarriesNoOwnCommitList`, `TestDescribeScopeCountsTheOwnCommits`, `TestReviewRangeRefusesWhenTheOwnCommitsCannotBeListed`
- [x] AC5 -> `TestOwnCommitsRecoversHarness142` (skips only on a clone without those commits)

## Test status

- `cd cli && go test ./...`: every package ok. `go vet ./...` and `GOOS=windows go vet ./...` clean; `golangci-lint run ./...` and `GOOS=windows golangci-lint run ./...`: 0 issues.
- Manual smoke test: `go run ./cmd/dotf spec review HARNESS-158-review-scope --dry-run --foreground` on this branch prints `Scope: git diff 8b2d0902…...HEAD (4 of 4 commits are the spec's own)`, with no `folder rule only` note, so the issue matched; the printed prompt carries `or reference #1727`. The match itself is pinned by `TestReviewRangeMatchesTheSpecsOwnIssue`, which drives `reviewRange` over a real `proposal.md` and origin.
- f5 (`TestOwnCommitsRecoversHarness142`) skips on a clone without the pinned commits, so its verification command requires `--- PASS` in `-v` output rather than trusting exit 0.
- No regressions: the launcher tests that stub the base with fake SHAs now stub the own-commit listing through the same kind of seam (`ownCommitsOf`).

## Archive review dispositions

Review: `review.md`, PASS, nan/qwen3.8-flash on `150a02b0`.

- F1 (Minor, REAL), `--full-history` load-bearing for no test and its comment inaccurate: applied in the archive PR. The flag is dropped: default history simplification already lists a slice on a side branch merged `--no-ff`, and leaves out one whose change the merge discarded (`-s ours`), which is not part of the diff. `TestOwnCommitsListsASideBranchSliceButNotADiscardedOne` pins both, and with the flag it fails (the discarded slice is listed). The comment now says why default simplification is kept. The all-commits query gained `--topo-order`, since commits sharing a timestamp came back in either order.
- F2 (Minor, THEORETICAL), `issue: "#0"` disables the issue rule without a reason: declined, the claim does not hold. `refFromMatch` rejects a number that is not positive, so `ResolveIssueLink` errors and the launch line prints `folder rule only: the issue link does not parse: issue number "0" is not a positive integer`. A `#0` row in `TestSpecIssueNumberFallsBackToTheFolderAndSaysWhy` now pins that disclosure.
- Question, `N of M` counts non-merge commits only: no change; the proposal's format and the prompt both say so.
- `tasks.md` closing boxes unticked: declined. `tasks.md` is in the contract set the review signed; the evidence those boxes name is recorded in this file and in #3279.

## Decisions made during implementation

- HARNESS-142 measured before writing the rule: its closing review (`review-request.json`: base `adf49c93`, reviewed `c4d0e6c8`) ran over 14 commits, of which 5 were its own, and the folder rule alone recovers all five. #3221 fixed HARNESS-142's feature without touching the folder or naming #1649, but it merged after that review, so a correct scope excludes it; the rule was not widened to every issue the spec names, which would have pulled in prior-art numbers.
- The list is advice in the prompt, not a field in `review-request.json`: the archive gate keeps checking the base SHA, so no stored contract changed.
- The issue number is matched only when the issue lives in the repository's own origin. Commit messages say `#N` for that repository's numbers, so a cross-repo issue would match unrelated commits.
- The issue rule sees a slice only if its squash message carries the reference. None of HARNESS-142's five slices mention #1649, while #3221's squash kept `Closes #3220`, so a slice PR that does not touch the folder needs `Part of #N` in its body (as #3279 does).
- #3221 changed HARNESS-142's reviewed feature after the closing review and before archive, without touching a contract file, so the digest gate stayed fresh. Not a gap: the fix carried its own PR review, and a review that predates a later fix is the expected order for a bug found after review.
- A git failure while listing refuses the launch instead of launching with an empty list, which would read as a spec with no commits of its own.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-571-a-negated-closing-keyword-still-closes-the-issue.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the review contract and the archive gate are unchanged; the decisions are local to the launcher and logged above
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: the review launcher exists in this repository only

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/HARNESS-158-review-scope/` -> `specs/archive/HARNESS-158-review-scope/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
