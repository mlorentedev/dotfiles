---
tags: [spec, tasks, templates]
created: "2026-08-30"
---

# Tasks - HARNESS-105

## Setup

- [x] Branch created from main: `feat/HARNESS-105`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC1] Write failing test for `cli/internal/shellsafe/quote_test.go` checking Bash and PowerShell quoting behaviors.
- [x] [AC1] Implement `cli/internal/shellsafe/quote.go` to make it pass.
- [x] [AC2] Refactor `cli/internal/spec/review_launch.go` and `cli/internal/initrepo/github_test.go` to use `shellsafe.Bash`.
- [x] [P] [AC3] Write failing test for `TerminalFailure` structured JSON output in a new file `cli/internal/errors/latch_test.go`.
- [x] [AC3] Implement `cli/internal/errors/latch.go` defining the `TerminalFailureError` and its JSON payload struct.
- [x] [AC4] Hook the top-level error handler in `cli/main.go` to intercept `TerminalFailureError` and output the JSON.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [x] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder
