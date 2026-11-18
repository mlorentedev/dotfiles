---
spec: "WIN-015-doctor-batch-launcher"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "382ad32a85826f8984c7f95b54b5f57709ca7f82"
reviewer: "agy/gemini-4-hydrogen-high"
date: "2026-11-17"
---

## Adversarial review

**Scope**: WIN-015-doctor-batch-launcher
**Sources**: `specs/WIN-015-doctor-batch-launcher/{proposal,tasks,verification,features.json}`, `git diff d24efa2ce48a817d259ed61c0d74ef3ff8cc95a5...HEAD` (spec commits `591adb75619e`, `382ad32a8582`; intermediate main commit `218a058d` from #3593 excluded from WIN-015 scope)

### Spec and task alignment
- **AC1 (`PATH` native executable -> OK)**: Implemented in `cli/internal/doctor/checks_agent_launchers.go` (`target == onPath` calls `rep.Pass`). Covered under `--verbose` by `TestCheckAgentLaunchers_ReportsWhatALaunchStarts` (`claude.exe`); non-verbose behavior is handled by `Report` (`TestReportSuppressesPassesUnlessVerbose`, `TestReport_Golden`), though see Finding 2 on mixed-section summary wording.
- **AC2 (`PATH` batch file with native `.exe` behind it -> OK naming both)**: Implemented via `lane.NativeTarget` and `rep.Pass` naming both `target` and `onPath`. Verified by `TestCheckAgentLaunchers_ReportsWhatALaunchStarts` (`pi.cmd` + `pi.exe`).
- **AC3 (batch-only launcher -> WARN naming file and `mise use -g <agent>`, never naming rejected tool-manager stub)**: Implemented in `case err != nil` using `rep.Warn`. Verified by `TestCheckAgentLaunchers_ReportsWhatALaunchStarts` (`opencode.cmd` + extensionless `stub` rejected by `lane.NativeTarget` and absent from output; `copilot.bat`). See Finding 1 on the warning's `dotf spec review` claim for `agy`.
- **AC4 (uninstalled -> SKIP; every `lane.Agents()` entry has a line under `--verbose`)**: Implemented by looping `lane.Agents()` and calling `rep.Skip(agent + ": not installed")` when `sys.LookPath` fails. Verified by `TestCheckAgentLaunchers_ReportsWhatALaunchStarts`, `TestCheckAgentLaunchers_CoversTheLaunchTable`, and `TestCheckAgentLaunchers_CoversTheReviewRunners` (AST-verified against `cli/internal/spec/review_launch.go`; mutation-tested by adding a dummy `"codex"` case and observing failure).
- **AC5 (off Windows -> one skipped line, neither `PATH` nor tool manager asked)**: Implemented (`if sys.GOOS != "windows"`). Verified by `TestCheckAgentLaunchers_SkipsOffWindows` (mutation-tested by removing `return` and observing failure).
- **AC6 (no process started via `System`; tool manager asked only via `System.NativeBehind`, once per batch agent; nil seam reads as none)**: Implemented via `System.NativeBehind` (`lane.MiseNative` in `realSystem()`). Verified by `TestCheckAgentLaunchers_AsksOnlyTheSeam` (mutation-tested by calling `lane.MiseNative` directly and observing 3 failures) and `TestCheckAgentLaunchers_WithoutAToolManagerAnswer`.
- **`tasks.md` & `features.json`**: All implementation boxes `[x]` match the diff; `tasks.md` accurately leaves `[ ] Lint passes` unchecked locally due to CRLF checkout (`golangci-lint run ./internal/doctor/...` fails only on `gofmt` for CRLF; LF-normalized `gofmt -d` is empty and `gocyclo`/`forbidigo` report 0 issues). No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain. `dotf spec digest WIN-015-doctor-batch-launcher` matches `review-request.json` `contract_digests`.
- **Test runs**: All 6 `TestCheckAgentLaunchers_*` tests, all 4 `features.json` verification commands, `TestSweep_*`/`TestRun_*`/`TestReport_Golden`, `go test ./internal/lane/... -run 'TestNativeTarget|TestRefuseBatchShim|TestExec_|TestAgents'`, `go test ./internal/spec/... -run 'TestReviewerCommand|TestResolveReviewer'`, and `go vet ./internal/doctor/` (Windows and `GOOS=linux`) passed in this session. Full `go test ./internal/doctor/ -count=1` failed only on the 4 pre-existing environment-dependent tests noted in `verification.md` (3 `TestCheckMiseTools_*` symlink privilege failures, `TestCheckWorkstationTarget_ByStatus` domain account).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | spec-launch-parity | `checkAgentLaunchers` unconditionally warns `"so dotf lane and dotf spec review refuse to start it"`, but `dotf spec review` does not refuse a batch shim for `agy` (`auth: "login"` in `harness/reviewer-pool.json:84` skips `dotf secrets run` in `cli/internal/spec/review_launch.go:144-150`, and `spec.RunWithLimits` in `cli/internal/spec/deadline.go:112` calls `exec.Command` directly without `lane.NativeTarget`/`RefuseBatchShim`). Also `claude`/`copilot`/`opencode` are not `spec review` runners (`review_launch.go:203-205`). On this host `where.exe agy` resolves `agy.exe`, so the mismatch is theoretical today. | Code inspection of `cli/internal/doctor/checks_agent_launchers.go:49-51`, `cli/internal/spec/review_launch.go:144-206`, `cli/internal/spec/deadline.go:112`, `harness/reviewer-pool.json:84`; `where.exe agy` returned `agy.exe`. | UNTESTED | code + tests (disposition in `verification.md`; either enforce `lane.NativeTarget` for login runners in `spec` or scope the warning text in `checks_agent_launchers.go`) |
| Minor | REAL | verification-staleness | `proposal.md:23` (`## What`) and `verification.md:26` state skipped lines print only with `--verbose` and passing checks collapse into a section summary alongside warnings, and `verification.md:21-24` pastes an outdated warning string (`"the only launcher is the batch file ..."`). In `cli/internal/doctor/report.go:124-160` and `cli/internal/doctor/testdata/report-plain.golden:5-14`, `StatusSkip` always prints without `--verbose`, mixed sections (`secPrinted > 0`) emit no `(N checks, all ok)` section summary (passing checks only increment `Results:`), and `checks_agent_launchers.go:49-51` emits `"PATH resolves it to the batch file ... and no native executable was found behind it"`. Additionally, `checks_agent_launchers_test.go` only tests `verbose = true`. | Observed in `cli/internal/doctor/report.go:121-163`, `cli/internal/doctor/testdata/report-plain.golden:5-14`, `cli/internal/doctor/checks_agent_launchers.go:49-51`, `specs/WIN-015-doctor-batch-launcher/verification.md:21-26`. | `TestReportSuppressesPassesUnlessVerbose`, `TestReport_Golden` (`Report` only); `checkAgentLaunchers` with `verbose=false` is UNTESTED | `verification.md` + tests (refresh pasted output and clarify `Report` mixed-section/skip behavior in `verification.md`; optionally add a `!verbose` test case) |
| Minor | THEORETICAL | lookup-consistency | `checkAgentLaunchers` calls `sys.LookPath(agent)` twice per installed agent (`checks_agent_launchers.go:37` and inside `lane.NativeTarget` at line 42). If `sys.LookPath` returned different paths across calls, `target == onPath` and the `"behind the batch file %s"` message would pair mismatched paths. | Code inspection of `cli/internal/doctor/checks_agent_launchers.go:37-56` and `cli/internal/lane/exec_native.go:36-44`. | UNTESTED (`launcherSys.LookPath` is static) | code + tests (pass single-lookup closure returning `onPath` into `lane.NativeTarget`) |
| Minor | THEORETICAL | wiring | `realSystem().NativeBehind` (`cli/internal/doctor/system.go:272`) has no unit test asserting it is non-nil (all `checks_agent_launchers_test.go` tests use `launcherSys` or nil `NativeBehind`). | Code inspection of `cli/internal/doctor/system.go:272` and `cli/internal/doctor/checks_agent_launchers_test.go`. | UNTESTED | tests (assert `realSystem().NativeBehind != nil`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All 6 ACs work and pass mutation checks; `agy` in `spec review` lacks `NativeTarget` parity (theoretical) and non-verbose mixed/skip wording in spec/verification differs from `Report`. |
| Verification       | B | Reproducible `go test` and `go vet` pass for all new/related targets, but `verification.md` quotes a pre-final warning string and misstates `Report` mixed-section summary collapse. |
| Scope              | A | Diff is strictly scoped to `checkAgentLaunchers`, `System.NativeBehind`, `sweep`, and `WIN-015-doctor-batch-launcher` artifacts. |
| Reliability        | B | Off-Windows short-circuits safely and nil `NativeBehind` degrades cleanly; duplicate `sys.LookPath` call per agent and unguarded `agy` `spec review` batch path are minor/theoretical gaps. |
| Maintainability    | A | `checkAgentLaunchers` is 33 lines, CC 7, nesting depth 2, reuses `lane.NativeTarget`/`lane.Agents()`, and passes `golangci-lint` (aside from CRLF `gofmt` checkout). |
| Handoff-readiness  | B | Spec docs have no draft tags and `tasks.md` tracks the local lint caveat; `verification.md` output should be refreshed for the final warning wording and `Report` behavior. |

### Verdict
PASS WITH GAPS

- **Disposition (Major | THEORETICAL | spec-launch-parity)**: Disposition in `specs/WIN-015-doctor-batch-launcher/verification.md` — either file a follow-up ticket to enforce `lane.NativeTarget` for login-auth runners in `spec.RunWithLimits` (or `ReviewerCommand`), or accept/narrow the warning text in `cli/internal/doctor/checks_agent_launchers.go`.
- **Disposition (Minor | REAL | verification-staleness)**: Refresh `specs/WIN-015-doctor-batch-launcher/verification.md` with the current warning wording (`PATH resolves it to the batch file ...`) and note that `Report` prints `[SKIP]` lines even without `--verbose` and counts hidden `[ OK ]` lines in `Results:` when warnings/skips are present; optionally add a non-verbose test in `checks_agent_launchers_test.go`.
- **Disposition (Minor | THEORETICAL | lookup-consistency)**: Track or optionally pass a single-lookup closure into `lane.NativeTarget` in `cli/internal/doctor/checks_agent_launchers.go`.
- **Disposition (Minor | THEORETICAL | wiring)**: Track or optionally assert `realSystem().NativeBehind != nil` in `cli/internal/doctor/checks_agent_launchers_test.go`.

### Recommended next steps
- Disposition the four findings above in `specs/WIN-015-doctor-batch-launcher/verification.md` (applied outside the contract set, ticketed, or declined with a reason) — do **not** edit `proposal.md`, `tasks.md`, or `features.json`, as that would invalidate this review's `contract_digests`.
- Refresh the pasted `dotf doctor` output in `specs/WIN-015-doctor-batch-launcher/verification.md` to match `checks_agent_launchers.go:49-51` and the actual `Report` non-verbose behavior.
- Confirm CI's `lint` job is green on LF checkout (since local `gofmt` fails only on CRLF).
- `dotf spec archive WIN-015-doctor-batch-launcher` is **advisable** once the findings are dispositioned in `verification.md` and CI `lint` is confirmed green.
