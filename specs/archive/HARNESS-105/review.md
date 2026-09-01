---
spec: "HARNESS-105"
verdict: "FAIL"
reviewed_sha: "dd47c5d0110d880b85c7943ce733b8ab8240d595"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-08-30"
---
## Adversarial review

**Scope**: HARNESS-105
**Sources**: specs/HARNESS-105/{proposal,tasks,verification}.md, git diff origin/main...HEAD

### Spec and task alignment
- `proposal.md` requires 100% test coverage for `shellsafe`, replacing ad-hoc strings in `review_launch.go`, and printing a JSON latch on `TerminalFailure`.
- `tasks.md` claims all are met.
- The diff shows `shellsafe` tests are implemented and `main.go` uses `IsTerminalFailure`.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker  | REAL    | CLI error handler | Wrapped `TerminalFailure` instances print the wrapper string before the JSON latch, breaking the strict prefix parsing orchestrators expect. | `fmt.Errorf("failed: %w", err)` causes `err.Error()` in `main.go` to print `failed: GENTLE_AI_SDD_FAILURE ...` rather than starting with the latch. | UNTESTED | code (fix `main.go` to unwrap using `errors.As` before printing `tfe.Error()`) + tests (add a test verifying wrapped error behavior) |
| Minor    | THEORETICAL | API design | The `reason` field in `TerminalFailureError` is unexported and has no getter, preventing programmatic access to the reason string outside the `errors` package without JSON unmarshaling. | Code review of `cli/internal/errors/latch.go` | UNTESTED | code (add `Reason()` string method) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | The core `TerminalFailure` formatting works in isolation, but breaks when errors are wrapped (a standard Go idiom), leading to an invalid JSON latch. |
| Verification       | B | Evidence covers criteria and proves tests pass, but lacks a test for wrapped errors which is critical for error formatting. |
| Scope              | A | The diff strictly matches the proposal without scope creep. |
| Reliability        | C | The error formatting breaks if the error is wrapped, failing the primary goal of providing a structured, parseable output to the agent. |
| Maintainability    | B | Code is well-structured, but the `reason` field in `TerminalFailureError` is unexported without an accessor. |
| Handoff-readiness  | A | Spec updates and tasks are completed appropriately. |

### Verdict
FAIL

### Recommended next steps (before archive)
- Fix the `TerminalFailureError` printing logic in `cli/cmd/dotf/main.go`. Use `goerrors.As` to extract the exact `TerminalFailureError` and call `.Error()` on it directly, ignoring wrapper strings.
- Add a test in `cli/internal/errors/latch_test.go` or `cli/cmd/dotf/main_test.go` that wraps a `TerminalFailureError` and verifies the JSON latch prints exactly as expected.
- (Optional) Export the `reason` field or provide a getter `Reason()` on `TerminalFailureError` for better API design.
