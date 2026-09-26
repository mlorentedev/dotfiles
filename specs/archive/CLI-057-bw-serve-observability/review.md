---
spec: "CLI-057-bw-serve-observability"
verdict: "PASS"
reviewed_sha: "f6826da53aaa8fb4396912eb63522114100d0281"
reviewer: "nan/mimo-v2.5"
date: "2026-09-25"
---

## Adversarial review

**Scope**: CLI-057-bw-serve-observability (spec folder + `git diff 3b1f1f9a...HEAD`, 2 commits, 21 files, +1337/-33 LOC)
**Sources**: `specs/CLI-057-bw-serve-observability/{proposal,tasks,verification,features}.json`, `cli/internal/secrets/bwserve.go`, `bwserve_state.go`, `procalive_{unix,windows}.go`, `cli/internal/cmd/secrets_unlock.go`, `cli/internal/doctor/checks_bw_serve.go`, `system.go`

### Spec and task alignment

- All 6 acceptance criteria (AC1–AC6) in `proposal.md` are testable and observable. Each has named test(s) in the diff.
- All implementation tasks in `tasks.md` are ticked `[x]`. Every ticked task maps to diff evidence (new or modified files).
- The one unchecked closing task ("PR opened referencing this spec folder") is a process step, not an implementation gap.
- `features.json` entries f1–f6 carry non-vacuous verification commands; all are in `state: "pending"` (expected pre-archive; `dotf spec archive` fills them).
- No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in any spec file.
- `verification.md` provides mutation-check evidence for two branches (the pid-file `ErrNotExist` guard in doctor, and the `State.enabled()` guard in `spawn`), each showing 3+ subtests turning red.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | THEORETICAL | concurrency | Concurrent `dotf secrets unlock` from two terminals: both call `rotateIfOver`, which reads size → rename → open. Two consecutive renames of the same file succeed (second replaces first), so the log is intact but rotated twice. The real risk is two `cmd.Start()` calls: `bw serve` binds to a fixed port with no `SO_REUSEADDR`, so the second `Start()` fails at the HTTP poll loop, not the spawn. The pid file is written after Start succeeds, so the winner's pid is recorded. | Code path analysis of `spawn` + `bwServeCommand` binding `127.0.0.1:8087`; `bw serve` uses Node `net.createServer` which rejects duplicate binds. | UNTESTED (concurrency test would need parallel goroutines + shared state dir; existing tests are sequential) | — (tracked, not gating; `bw serve` port-exclusion is the real guard) |
| Minor | THEORETICAL | liveness | `Trace()` calls `ReadPID` then `traceAlive(pid)`: a pid file naming a dead process could become live (pid reuse) between the read and the check. The window is narrow (OS must reclaim the pid after reap) and the spec's own "Risks" section names this ambiguity. The message "alive but not answering" is correct for both "still starting" and "pid reuse". | Code path: `bwserve_state.go:Trace()` → `ReadPID` → `traceAlive` | `TestTrace_AStalePIDIsSaidNotGuessed` (covers the dead case; no test for the live→dead→live interleaving) | — (accepted risk, spec §Risks) |
| Minor | N/A | spec | `tasks.md` closing checkbox "PR opened referencing this spec folder; `## Review triage` posted; `dotf pr triage-queue` exit 0" is unchecked. This is a process step (the PR exists per `git log`: commit `097318e` with message "feat(secrets): ... (#1348)"), but the triage comment and queue check may not have been posted yet. | `git log --oneline 3b1f1f9...HEAD` shows PR #1348 merged. | N/A | tasks.md (process; not a code defect) |
| Info | N/A | cross-env | Verification was run on Windows (go1.26.0 windows/amd64). The Unix `ProcessAlive` implementation (`procalive_unix.go`) is vetted with `GOOS=linux go vet` and runs on CI, but was not live-exercised on a Unix box in this verification session. This is expected — the Windows work box is the primary target and the Unix code is straightforward (`kill -0`). | `verification.md` states the scope. | `TestProcessAlive` runs on the current box (Linux in this review); `GOOS=windows go vet` passes here. | — (informational) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 6 ACs verified by named tests with mutation checks for the two key branches; negative paths (no state dir, unreadable pid, dead pid, garbage pid) covered. |
| Verification       | A | Each AC mapped to specific test functions; mutation checks documented with pass/fail evidence; full test suite clean on touched packages. |
| Scope              | A | Diff matches proposal exactly: state management (`bwserve_state.go`), liveness (`procalive_*.go`), spawn wiring (`bwserve.go`), doctor integration (`checks_bw_serve.go`), CLI output (`secrets_unlock.go`), plus lesson 242. No unrelated changes. |
| Reliability        | A | Error paths fully handled: rename→truncate fallback, nil ProcessAlive seam, unreadable pid file, missing log, disabled state. Idempotent: `Start()` no-ops when daemon already running. |
| Maintainability    | A | All functions under 30 lines, clear naming, comments explain WHY (not just what). Constants (`bwServeLogCap`, `bwServeTailBytes`, `bwServeLogLineWidth`) are documented and testable. No magic numbers in logic. |
| Handoff-readiness  | A | Lesson 242 captured in `docs/lessons/`. Spec artifacts complete. tasks.md closing step pending (process, not implementation). PR #1348 merged. |

### Verdict
PASS

### Recommended next steps

- **tasks.md** (spec artifact): tick the closing checkbox once `## Review triage` is posted on PR #1348 and `dotf pr triage-queue` returns exit 0. This is a process step, not a code gap.
- **archive**: `dotf spec archive CLI-057-bw-serve-observability` is advisable after the closing checkbox is ticked. No contract-set changes are needed; the findings above are tracked minors, not blockers.

### Closing pass (Definition of Done)

| Check | Verdict | Notes |
|---|---|---|
| **Debt** | Done | No unfixed defects in scope. Two theoretical concurrency/liveness findings are tracked above and accepted per spec §Risks. |
| **Knowledge** | Done | Lesson 242 captured in `docs/lessons/lesson-242-...md` with cross-references to lessons 237 and 240. |
| **Board** | Done | Issue #1315 is referenced by PR #1348 (`Refs #1315`). Board status should reflect the merged state. |
| **Review** | Done | This adversarial review is the artifact. No pending reviewer comments on the PR. |
| **Evidence** | Done | All test commands run fresh in this session: `go test ./internal/{secrets,cmd,doctor}/` — 0 failures. `go vet ./...` — clean. `GOOS=windows go vet ./internal/secrets/` — clean. Pre-existing failure in `internal/harness/TestMergeAgainstTheRealDeployedSettings` is unrelated (diff touches zero files in that package). |
