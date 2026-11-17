---
tags: [spec, tasks]
created: "2026-11-11"
id: "BUG-166-review-base-widen"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-166-review-base-widen

## Setup

- [x] Branch created from main: `fix/review-base-range`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] Failing test: the #3258 shape (base, fix, spec+tests) resolves to the commit before the fix, with an `origin/main` ref at the fork point
- [x] [AC1] Implement the widen-only fork point in `ResolveReviewBaseDetail`, recording the traced base as `WidenedFrom`
- [x] [AC2] Tests: squash on main, main merged into the branch, no `origin/*` ref: base equals the traced base
- [x] [AC3] Existing refusal and rename tests pass unchanged
- [x] [AC1] Failing test: commits in fork..traced base count as own even without an issue reference; implement in `OwnCommits`
- [x] [AC4] `describeScope` and the prompt say the base was widened and from which commit; launcher test

## Closing

- [x] Every acceptance criterion is covered by a test
- [x] `features.json` entries carry non-vacuous verification commands
- [x] `go build ./... && go vet ./... && go test ./...`, `golangci-lint run`, `GOOS=windows go vet ./...`
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder
