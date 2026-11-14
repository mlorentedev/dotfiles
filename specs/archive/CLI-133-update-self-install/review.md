---
spec: "CLI-133-update-self-install"
verdict: "PASS"
reviewed_sha: "d738f8bb673118664730370a19601a0c5940dd40"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-12"
---

## Adversarial review

**Scope**: CLI-133-update-self-install
**Sources**: specs/CLI-133-update-self-install/{proposal,tasks,verification}.md, git diff 35bdbe72d0c364fd102040f8b5a3b5cffe102f01...HEAD

### Spec and task alignment
The diff spans commits `3e284ffc46cc`, `d6ce28f543d5`, and `d738f8bb6731` on the feature branch (merging `origin/main` at `31a4dd6a` after #3428 landed).
- Acceptance criteria AC1 through AC6 from `proposal.md` map directly to verified tests across `internal/tools/`, `internal/update/`, and `internal/cmd/`.
- Tasks in `tasks.md` accurately correspond to the code changes. The closing tasks (`go test`, linting, runbook update) have been independently verified in this review.
- The BUG-172 (`#3428`) `InstallerConverge` report reader seam was cleanly removed in favor of direct error distinction between Go installation (`install-failed`) and subprocess convergence (`converge-failed`), as specified in proposal Section What 3 and verification Decision 4.
- `update.Run` was refactored down to 23 lines (closing #3427), well below the 40-line threshold.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Minor | THEORETICAL | resilience / race | `replaceFile` executes rename-aside (`dst -> dst.old`, then `staged -> dst`) on every OS without a `GOOS` branch; on POSIX systems (Linux/macOS), this introduces a microsecond window where `dst` does not exist on disk, unlike an atomic `os.Rename(staged, dst)`. | Code inspection of `cli/internal/tools/install.go:786-805` | `TestPlaceBinaryReplacesARunningExecutable` | code (`cli/internal/tools/install.go`) |
| Minor | THEORETICAL | resilience / win32 | If a long-running process on Windows holds `dst.old` open across multiple update cycles, `os.Remove(aside)` will fail with Access Denied, aborting subsequent replacements. | Code inspection of `cli/internal/tools/install.go:788-790` | `TestPlaceBinaryReplacesARunningExecutable` | code (`cli/internal/tools/install.go`) |
| Minor | THEORETICAL | cli / I/O | `convergeWith` connects stdout and stderr to Cobra's outputs, but does not assign `cmd.Stdin = c.InOrStdin()`. While self-update is designed as an unattended scheduled run, interactive invocations cannot pipe stdin to child convergence. | Code inspection of `cli/internal/cmd/update.go:212-217` | UNTESTED | code (`cli/internal/cmd/update.go`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | A | All 6 acceptance criteria verified with comprehensive positive and negative test coverage. |
| Verification | A | Test suite passes, linter and cross-OS vet clean, and red-green mutation testing confirmed. |
| Scope | A | Scope matches proposal exactly: Go self-install primitive, update rewiring, and runbook updates without creep. |
| Reliability | A | Full verification pipeline (sha256, staged probe, atomic rename-aside restore, offline skip) prevents broken binary state. |
| Maintainability | A | Functions under 24 lines, cyclomatic complexity <= 7 across all new code, zero lint issues. |
| Handoff-readiness | A | Spec triad and features.json match code, runbook updated, clear disposition guidance provided. |

### Verdict
PASS

### Recommended next steps
The contract set (`proposal.md`, `tasks.md`, `features.json`) is closed; do not edit those files as doing so would invalidate this passing review.
1. In `verification.md` (which is outside the staleness check set), disposition the minor theoretical findings:
   - Accept the uniform cross-OS rename-aside in `replaceFile` as an intentional design choice avoiding OS-specific branches.
   - Accept the Windows `aside` removal behavior as a safe fail-closed lock when an old process remains active.
   - Note the non-interactive stdin behavior for `convergeWith` as standard for scheduler-invoked commands.
2. Complete the promotion candidate check in `verification.md` (e.g. repo lesson for running-binary replacement on Windows).
3. Proceed with spec archive (`dotf spec archive CLI-133-update-self-install`).
