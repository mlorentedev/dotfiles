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
- Manual smoke test: `go run ./cmd/dotf spec review HARNESS-158-review-scope --dry-run --foreground` on this branch prints `Scope: git diff 8b2d0902…...HEAD (1 of 1 commits are the spec's own)`, and its prompt names `#1727` as the matched issue.
- No regressions: the launcher tests that stub the base with fake SHAs now stub the own-commit listing through the same kind of seam (`ownCommitsOf`).

## Decisions made during implementation

- HARNESS-142 measured before writing the rule: its closing review (`review-request.json`: base `adf49c93`, reviewed `c4d0e6c8`) ran over 14 commits, of which 5 were its own, and the folder rule alone recovers all five. #3221 fixed HARNESS-142's feature without touching the folder or naming #1649, but it merged after that review, so a correct scope excludes it; the rule was not widened to every issue the spec names, which would have pulled in prior-art numbers.
- The list is advice in the prompt, not a field in `review-request.json`: the archive gate keeps checking the base SHA, so no stored contract changed.
- The issue number is matched only when the issue lives in the repository's own origin. Commit messages say `#N` for that repository's numbers, so a cross-repo issue would match unrelated commits.
- A git failure while listing refuses the launch instead of launching with an empty list, which would read as a spec with no commits of its own.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-158-review-scope/` -> `specs/archive/HARNESS-158-review-scope/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
