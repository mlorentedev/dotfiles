---
tags: [spec, tasks, review]
created: "2026-10-30"
---

# Tasks - CLI-132-root-commit-review

> TDD order. Run every command from `cli/`. Each behavior is observed failing
> before production code changes. `[AC<n>]` maps to `proposal.md`.
>
> Local verification uses `go test`, `go vet`, and `gofmt -l`. golangci-lint is
> not run on this workstation (endpoint security blocks it); CI runs it.

## Setup

- [x] Branch created: `feat/root-commit-review` (worktree from `origin/main`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"
- [x] Baseline: `go test ./internal/spec/ -count=1` passes

## Implementation

- [x] [P] [AC1] In `internal/spec/review_base_test.go`, add
  `TestResolveReviewBaseDetailReportsADirectRootCommitSpec`,
  `TestResolveReviewBaseDetailReportsASpecRenamedOutOfTheRootCommit`,
  `TestResolveReviewBaseDetailReportsAnUncommittedSpec`,
  `TestResolveReviewBaseDetailReportsAnUntraceableMove`, and
  `TestResolveReviewBaseDetailKeepsTheBaseOfAnOrdinarySpec`. Run
  `go test ./internal/spec/ -run TestResolveReviewBaseDetail -count=1`.
  Expected: FAIL, `undefined: ResolveReviewBaseDetail`.
- [x] [AC1] In `internal/spec/review_request.go`, add `BaseRefusal`,
  `ReviewBase`, and `ResolveReviewBaseDetail`; confirm a root commit with
  `git rev-list --parents -n 1`; make `ResolveReviewBase` return its `SHA`.
  Run `go test ./internal/spec/ -run TestResolveReviewBase -count=1`.
  Expected: PASS, including the existing rename regressions.
- [x] [P] [AC3] In `internal/spec/review_base_test.go`, add
  `TestEmptyTreeIsGitsEmptyTree` and
  `TestInitialScopeCoversTheRootImplementationAndLaterCorrections`. Run
  `go test ./internal/spec/ -run 'TestEmptyTree|TestInitialScope' -count=1`.
  Expected: FAIL, `undefined: EmptyTree`.
- [x] [AC3] Add `EmptyTree(repoRoot)` using `git hash-object -t tree --stdin`
  with empty input. Rerun the same command. Expected: PASS.
- [ ] [P] [AC3] In `internal/spec/review_launch_test.go`, add
  `TestReviewPromptStatesTheInitialScope` (empty tree, root commit,
  `git diff <tree> HEAD`, no `<tree>...`) and
  `TestReviewPromptKeepsTheTripleDotScopeForAnOrdinaryBase`. Run
  `go test ./internal/spec/ -run TestReviewPrompt -count=1`.
  Expected: FAIL, `undefined: ReviewScope`.
- [ ] [AC3] In `internal/spec/review_launch.go`, add `ReviewScope` and
  `ReviewPromptForScope`; make `ReviewPromptIn` delegate with an ordinary
  scope. Rerun. Expected: PASS, existing prompt tests unchanged.
- [ ] [P] [AC4] In `internal/spec/review_request_test.go`, add
  `TestReviewRequestRecordsTheInitialScope` (`scope`, `root_sha`, `base_tree`
  present; `base_sha` absent). Run
  `go test ./internal/spec/ -run TestReviewRequestRecords -count=1`.
  Expected: FAIL, `undefined: WriteReviewRequestScoped`.
- [ ] [AC4] Add `Scope`, `RootSHA`, and `BaseTree` (omitempty) to
  `ReviewRequest`, and `WriteReviewRequestScoped`; make
  `WriteReviewRequestIn` delegate. Rerun. Expected: PASS.
- [ ] [P] [AC5] In `internal/spec/review_request_test.go`, add
  `TestProvenanceRefusesAnUnknownReviewScope`,
  `TestProvenanceRefusesAnInitialScopeWithoutItsRootOrTree`, and
  `TestProvenanceAcceptsAnInitialScope`. Run
  `go test ./internal/spec/ -run TestProvenance -count=1`.
  Expected: FAIL on the two refusals.
- [ ] [AC5] In `internal/spec/review.go`, add `checkReviewScope` and call it
  from `checkReviewProvenanceIn`. Rerun. Expected: PASS, existing provenance
  tests unchanged.
- [ ] [P] [AC2] [AC4] In `internal/cmd/spec_review_base_test.go`, change the
  `resolveReviewBase` seam helpers (here and in `makeRepo`) to return
  `spec.ReviewBase`, then add `TestSpecReviewInitialLaunchesForARootCommitSpec`
  (dry-run prints scope, tree, and root; writes no request; leaves `review.md`
  untouched), `TestSpecReviewInitialRefusesEveryOtherBaseState` (resolved, not
  committed, untraceable), and
  `TestSpecReviewWithoutInitialRefusesARootCommitSpecAndNamesTheFlag`. Run
  `go test ./internal/cmd/ -run TestSpecReview -count=1`.
  Expected: FAIL, `unknown flag: --initial`.
- [ ] [AC2] [AC4] In `internal/cmd/spec.go`, add `--initial`, make
  `reviewRange` return `spec.ReviewScope`, and pass it to the scoped prompt and
  request; document the flag in the command help. Rerun. Expected: PASS.
- [ ] Refactor only after green. `gofmt -l internal` prints nothing,
  `go vet ./internal/spec/ ./internal/cmd/` passes, and no touched
  `internal/cmd` function exceeds cyclomatic complexity 15.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one
  named test
- [ ] Every acceptance criterion has a matching entry in `features.json`
- [ ] `go test ./internal/spec/ ./internal/cmd/ -count=1` passes
- [ ] `gofmt -l internal` is empty and `go vet ./...` passes
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in with command output
- [ ] PR opened referencing this spec folder

## Machine-readable features

See `features.json` beside this file. Only the verification harness may move a
feature from `pending` to `passing`.
