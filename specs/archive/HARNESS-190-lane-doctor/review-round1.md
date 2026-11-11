---
spec: "HARNESS-190-lane-doctor"
verdict: "FAIL"
reviewed_sha: "18d713234efa06bb2a799422b873e697465c72b7"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-11"
---

## Adversarial review

**Scope**: HARNESS-190-lane-doctor
**Sources**: specs/HARNESS-190-lane-doctor/{proposal.md,tasks.md,verification.md}, git diff 658ef64b2a96f79ca6d54da2855c0aa122f1d799...HEAD

### Spec and task alignment
- **AC1 (Corrupted marker)**: Covered. `TestAuditLanes_ReportsACorruptedMarkerByField` tests nine malformed marker shapes (syntax errors, unknown keys, wrong version, empty lists, separators, drive prefixes, duplicates, differing set).
- **AC2 (Drift in both directions)**: Covered. `TestAuditLanes_ReportsDriftInBothDirections` tests manual rename (`mv`), `git worktree move`, undeclared records in `.git/worktrees/*/`, relative `gitdir` entries, and unreadable records.
- **AC3 (Broken lane reporting)**: Partially covered. Missing lease, missing identity record, lease lane mismatch, and plain directory (post-`git worktree remove`) are tested in `TestAuditLanes_ReportsABrokenLane`. However, an adversarial probe showed that replacing a lane with a plain directory *without* `git worktree remove` bypasses worktree validation and prescribes an unviable fix (`dotf lane init`), which aborts (see Finding 1).
- **AC4 (CLI contract of `dotf lane doctor`)**: Covered. `TestLaneDoctor_ExitsZeroWithAtMostOneLine` and `TestLaneDoctor_AFindingExitsOneWithoutCobrasErrorLine` pin silent exit 0 on coherent repo, single line on unmanaged/uninitialised repo, and exit 1 with ERROR/FIX pairs without Cobra's `Error:` line on finding.
- **Tasks and verification alignment**: All implementation tasks in `tasks.md` are marked `[x]` with commit SHAs matching git history. Pre-archive items in `## Closing` are pending. `features.json` f1..f4 are in `verified` state with valid test commands and captured evidence.
- **Contract digests**: Verified with `dotf spec digest HARNESS-190-lane-doctor`; output matches `specs/HARNESS-190-lane-doctor/review-request.json` exactly.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | lane-audit | In `auditLane`, when `len(records) == 1` and `samePath(r.path, expected)` holds, `assertWorktreeOf(absRepo, commonDir, expected)` is skipped. If a lane directory is wiped and replaced by a plain directory (or recreated via `mkdir` after manual deletion without `git worktree remove`), the git common dir `<repo>/.git/worktrees/<name>/` remains registered. `auditLane` directly calls `auditLease(name, expected)`, which reports a missing lease (`.dotf-worktree.json`) and prescribes `FIX: dotf lane init`. Running `dotf lane init` then immediately fails with `lane init: <expected> exists but is not a git worktree; move it aside, or run git worktree repair if it was one`. The doctor prescribes a repair command that cannot succeed. | Executed probe: wiping `feat-lane-secondary` and replacing with plain `mkdir` caused `lane.AuditLanes` to return `ERROR: feat-lane-secondary: has no lease... FIX: dotf lane init`. Calling `lane.InitLanes` immediately returned error: `lane init: ... exists but is not a git worktree`. | UNTESTED (`audit_test.go` only tested plain directory after `git worktree remove --force`, which pruned the admin dir and triggered `len(records) == 0`) | code (`cli/internal/lane/audit.go`) + tests (`cli/internal/lane/audit_test.go`) |
| Minor | REAL | code-quality | `AuditLanes` in `cli/internal/lane/audit.go` has a cyclomatic complexity of 18 (measured via `cyclop`) and a function length of 63 lines, exceeding the code quality threshold of CC < 10 (and < 15 for solid maintainability) and function length < 40 lines. | `golangci-lint run --enable cyclop ./internal/lane` reports `internal/lane/audit.go:46:1: calculated cyclomatic complexity for function AuditLanes is 18, max is 10 (cyclop)` | `TestAuditLanes_*` | code (`cli/internal/lane/audit.go`) |
| Minor | THEORETICAL | marker-validation | `validateNames` in `cli/internal/lane/audit.go` checks for separators (`/\`), drive prefixes, and empty/duplicate entries, but does not reject special directories like `.` or `..` in `third_party_roots`, nor does it reject entries containing leading or trailing whitespace. | Code inspection: `strings.ContainsAny(n, "/\\")` is false for `.` and `..`, and `strings.TrimSpace(n) == ""` is false for `"   entry   "`. | UNTESTED | code (`cli/internal/lane/audit.go`) + tests (`cli/internal/lane/audit_test.go`) |
| Minor | THEORETICAL | marker-validation | `parseMarker` uses `json.NewDecoder(bytes.NewReader(raw)).Decode(&m)` without checking for trailing non-whitespace data after the first JSON object, allowing trailing syntax or garbage to pass undetected. | Standard Go `json.Decoder.Decode` consumes the first valid JSON token/object and stops, ignoring subsequent trailing bytes. | UNTESTED | code (`cli/internal/lane/audit.go`) + tests (`cli/internal/lane/audit_test.go`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Happy path and specified cases pass, but `auditLane` misdiagnoses a plain directory at a registered lane path as a missing lease and prescribes an unviable fix (`dotf lane init`). |
| Verification       | A | Red-green TDD commits, comprehensive test coverage for all 4 ACs, and verified `features.json` entries with reproducible commands. |
| Scope              | A | Clean diff bounded to 254 executable LOC (under 300 LOC cap); no unrelated changes or scope creep. |
| Reliability        | B | Purely diagnostic command (no destructive auto-repair); error handling is robust across Git and OS boundaries. |
| Maintainability    | C | `AuditLanes` cyclomatic complexity is 18 (> 15) and 63 lines (> 40 lines). |
| Handoff-readiness  | A | Clear runbook section 7.6, lesson 581 captured, ADR-061 updated, and follow-up issue #3327 filed. |

### Verdict
FAIL

### Recommended next steps
1. **Fix `auditLane` worktree validation (code)**: In `cli/internal/lane/audit.go` (`auditLane`), in `case len(records) == 1:` when `samePath(r.path, expected)` matches, call `assertWorktreeOf(absRepo, commonDir, expected)` before calling `auditLease(name, expected)`. If `assertWorktreeOf` returns an error, return `Finding{name, err.Error(), "move the directory aside, then dotf lane init"}`.
2. **Add regression test (tests)**: In `cli/internal/lane/audit_test.go`, add a test case in `TestAuditLanes_ReportsABrokenLane` verifying that a plain directory at a lane path whose git admin directory is still intact is reported as "not a git worktree" rather than "has no lease".
3. **Refactor `AuditLanes` (code)**: Extract initialisation check and undeclared record scanning into small helpers to bring `AuditLanes` cyclomatic complexity <= 10 and length < 40 lines.
4. **Re-run adversarial review**: Once the fix is applied and tests pass, re-run `dotf spec review HARNESS-190-lane-doctor`.
