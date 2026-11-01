---
tags: [spec, verification, review]
created: "2026-10-30"
---

# Verification - CLI-132-root-commit-review

## Evidence

Run every command from `cli/`. Commits are on `feat/root-commit-review`.

- [x] **AC1** -> `3fdded8e`: `TestResolveReviewBaseDetailReportsADirectRootCommitSpec`,
  `TestResolveReviewBaseDetailReportsASpecRenamedOutOfTheRootCommit`,
  `TestResolveReviewBaseDetailReportsAnUncommittedSpec`,
  `TestResolveReviewBaseDetailReportsAnUntraceableMove` (both untraceable
  shapes), `TestResolveReviewBaseDetailKeepsTheBaseOfAnOrdinarySpec`. The
  eleven existing `TestResolveReviewBase*` regressions pass unchanged.
- [x] **AC2** -> `d90643c1`: `TestSpecReviewInitialLaunchesForARootCommitSpec`,
  `TestSpecReviewInitialRefusesEveryOtherBaseState` (ordinary, uncommitted,
  untraceable; previous `review.md` and request byte-identical after the
  refusal), `TestSpecReviewWithoutInitialRefusesARootCommitSpecAndNamesTheFlag`.
- [x] **AC3** -> `3fdded8e`, `39245ab5`: `TestEmptyTreeIsGitsEmptyTree` (SHA-1
  and SHA-256 repositories), `TestInitialScopeCoversTheRootImplementationAndLaterCorrections`
  (also pins that `git diff <tree>...HEAD` fails), `TestReviewPromptStatesTheInitialScope`,
  `TestReviewPromptKeepsTheTripleDotScopeForAnOrdinaryBase` (ordinary prompt
  byte-identical to `ReviewPromptIn`).
- [x] **AC4** -> `39245ab5` and `d90643c1`: `TestReviewRequestRecordsTheInitialScope`,
  `TestRecordReviewRequestWritesTheInitialScope`, and the dry-run assertions in
  `TestSpecReviewInitialLaunchesForARootCommitSpec`.
- [x] **AC5** -> `39245ab5`: `TestProvenanceRefusesAnUnknownReviewScope`,
  `TestProvenanceRefusesAnInitialScopeWithoutItsRootOrTree` (no root, no tree,
  also a base), `TestProvenanceAcceptsAnInitialScope`; every existing
  `TestProvenance*` passes unchanged.

### Red before green

| Batch | Command | Red observed |
|---|---|---|
| AC1, AC3 resolver | `go test ./internal/spec/ -run 'TestResolveReviewBaseDetail\|TestEmptyTree\|TestInitialScope' -count=1` | build failed: `undefined: ResolveReviewBaseDetail` |
| AC3, AC4, AC5 | `go test ./internal/spec/ -run 'TestReviewPromptStatesTheInitialScope\|TestReviewPromptKeepsTheTripleDot\|TestReviewRequestRecordsTheInitialScope\|TestProvenance' -count=1` | build failed: `undefined: ReviewScope` |
| AC5 behaviour | same command after `ReviewScope`, before `checkReviewScope` | `TestProvenanceRefusesAnUnknownReviewScope` and all three `TestProvenanceRefusesAnInitialScopeWithoutItsRootOrTree` cases FAIL; `TestProvenanceAcceptsAnInitialScope` PASS |
| AC2, AC4 command | `go test ./internal/cmd/ -run 'TestSpecReview\|TestRecordReviewRequest' -count=1` | build failed: seam type mismatch, `undefined: emptyTreeOf` |

## Test status

- Focused: all commands in `features.json` pass.
- Full packages: `go test ./internal/spec/ ./internal/cmd/ -count=1 -p 1 -timeout 40m` -> PASS; see
  "Full-suite runs" below.
- `go build ./...` and `go vet ./internal/cmd/ ./internal/spec/`: clean.
- Formatting: `gofmt` reports no change on every touched `.go` file after
  normalizing the Windows checkout's CRLF to LF.
- golangci-lint was not run locally: endpoint security on this workstation
  blocks it. CI runs it. Touched `internal/cmd` functions stay well below the
  gocyclo line of 15 by inspection: `initialScope` 5, `reviewRange` 4,
  `noReviewBaseError` 2, `describeScope` 2; the `spec review` closure gains no
  branch.

### Full-suite runs

1. First run (607 s): `internal/cmd` failed `TestHelpTextHasNoInternalReferences`
   and `TestOutputStringsHaveNoInternalReferences` because the new `--help`
   example used an id with a spec-area prefix. Corrected to
   `FEAT-002-first-commit`; both tests pass.
2. Second run, both packages in parallel: each exceeded `go test`'s default
   10-minute timeout inside the `git` test helper. No assertion failed; every
   git call takes seconds on this workstation.
3. Third run, `go test ./internal/spec/ ./internal/cmd/ -count=1 -p 1 -timeout 40m`:
   `ok internal/spec 531.779s`, `ok internal/cmd 367.146s`.

### Manual smoke test (real root-commit history)

A binary built from this branch (`go build -o dotf-cli132.exe ./cmd/dotf`) was
run in a private repository whose spec was added in its root commit and renamed
later. Only `--dry-run` was used; no request, verdict, or transcript was
written, and the target checkout stayed clean (hashes of its `review.md` and
`review-request.json` identical before and after).

- Without `--initial`: exit 1, the usual refusal plus "This spec was first
  added in the root commit <root>: `dotf spec review <id> --initial` reviews it
  from the empty tree instead". `<root>` equals `git rev-list --max-parents=0 HEAD`.
- With `--initial --dry-run`: exit 0 and
  `Scope: initial review, git diff 4b825dc642cb6eb9a060e54bf8d69288fbee4904 HEAD (spec first added in root commit <root>)`.

## Decisions made during implementation

- `ResolveReviewBase` keeps its signature and answers; the new
  `ResolveReviewBaseDetail` carries the reason, so no existing caller or test
  changed meaning.
- A root commit is confirmed positively with `git rev-list --parents -n 1`;
  any other git failure reads as untraceable, never as a root, so a broken
  repository cannot widen a review to the whole tree.
- The empty tree is computed, not hard-coded, so SHA-256 repositories work.
- `ReviewPromptIn` and `WriteReviewRequestIn` delegate to scoped variants with
  an ordinary scope; ordinary prompts and requests are byte-identical to before.
- Archive provenance validates the scope right after the empty-field check, so
  a hand-written scope is refused before the sha and reviewer comparisons.
- The command seam `resolveReviewBase` now returns `spec.ReviewBase`, and
  `emptyTreeOf` joins it, so command tests can drive every resolver answer.

## Review dispositions (2026-11-01, nan/mimo-v2.6-flash, PASS)

- F1, the ordinary-scope prompt does not warn about merged-in work: ticketed as #3018. It is outside CLI-132's scope, because the proposal leaves ordinary behaviour unchanged.
- F2, `spec-gate` red while the spec is unarchived: applied, the spec is archived in this PR.
- F3, `--initial` combined with `--second` or `--fallback-reason` is untested: ticketed as #3019.
- F4, promotion candidates held placeholders: applied, answered below.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the two lost review rounds came from the review environment and are tracked in #2979 and on #1649; the change itself taught nothing new
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the opt-in `--initial` scope is recorded in proposal.md and keeps BUG-108's default refusal
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: specific to dotf's spec review

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-132-root-commit-review/` -> `specs/archive/CLI-132-root-commit-review/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
