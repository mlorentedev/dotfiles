---
spec: "PI-002-windows-fixes"
verdict: "FAIL"
reviewed_sha: "8bb09c2f63f5b83212ceb7904a19492b8f2c8633"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-10-03"
---
## Adversarial review

**Scope**: PI-002-windows-fixes
**Sources**: specs/PI-002-windows-fixes/{proposal.md,tasks.md,verification.md}, git diff c8ebfec16b5359f76e9c490a0c47f0f2649dcd23...HEAD

### Spec and task alignment
- `proposal.md` requires `CREATE_NO_WINDOW` (0x08000000) for `bw serve`, but implementation deliberately omitted it in favor of `HideWindow: true`. The spec was not updated.
- `proposal.md` AC "Next run of dotf doctor on Windows is clean" is unmet and un-ticked.
- `verification.md` lacks runtime evidence for both the Windows API change and the `dotf doctor` execution.
- Diff scope includes multiple unrelated commits (`FIX-WIN-TUI-001`, harness chores, releases).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker  | REAL    | `models.json` / Spec | AC "dotf doctor on Windows is clean" is unmet. Lowering deepseek limits to 128k/16k produces a limit drift warning (`contextWindow is 128000, below the provider's 163840`). | Ran `go run ./cmd/dotf doctor` and reproduced the warning. | UNTESTED | code + spec |
| Major    | REAL    | `bwserve_windows.go` / Spec | Spec vs code mismatch: Spec explicitly requires `CREATE_NO_WINDOW`, but code omits it (as `lesson-333` notes it's ignored when used with `DETACHED_PROCESS`). The contract set was not updated. | Code inspection vs `proposal.md` requirement. | UNTESTED | spec |
| Major    | REAL    | `verification.md` | The Windows process detachment fix was never executed. `verification.md` explicitly cites `go build` as the only proof. | `verification.md` states "Verified go build ... completes successfully". | `TestBWServeDetachAttr_ChildHasNoConsole` | tests + `verification.md` |
| Minor    | REAL    | PR / Diff Scope | Diff contains mixed, unrelated changes (e.g., `FIX-WIN-TUI-001`, `powershell/profile.ps1`, `versions.conf`). | `git log c8ebfec16b5359f76e9c490a0c47f0f2649dcd23..HEAD` shows mixed commits. | UNTESTED | vault |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | D | AC for `dotf doctor` is unmet; a warning is still emitted for `models.json`. |
| Verification       | D | No runtime evidence for the Windows syscall fix or the `dotf doctor` check; only a compile check. |
| Scope              | C | Significant unrelated changes (`FIX-WIN-TUI-001`, harness chores) mixed into the reviewed diff. |
| Reliability        | B | Changes do not introduce crashes; the API usage is correct as per Microsoft docs. |
| Maintainability    | B | Code is clear, complexity is low, and comments accurately explain the API quirk. |
| Handoff-readiness  | C | Lessons were captured (`lesson-333`), but spec updates for the API change were deferred. |

### Verdict
FAIL

### Recommended next steps
- Update `ai/pi/models.json` limits to match the provider catalog (163840 context / 16384 output) so `dotf doctor` runs cleanly, and check off the corresponding AC in `proposal.md`.
- Update `proposal.md` and `tasks.md` to remove the incorrect `CREATE_NO_WINDOW` requirement, reflecting the `HideWindow: true` solution.
- Provide runtime verification evidence for the Windows `bw serve` fix and `dotf doctor` in `verification.md`.
- Since this is a FAIL on the contract set, a re-review is required after the spec and code are reconciled. Archive is currently NOT advisable.
