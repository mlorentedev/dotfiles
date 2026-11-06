---
spec: "BUG-164-copilot-settings-guard"
verdict: "PASS"
reviewed_sha: "273da959b086540fe4a803fa81c659bf02f1166f"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-05"
---

## Adversarial review

**Scope**: BUG-164-copilot-settings-guard
**Sources**: `specs/BUG-164-copilot-settings-guard/{proposal,tasks,verification}.md` + `git diff c4d0e6c8b891159a2fb7267140d76a7fa9421689...HEAD` (PR #3218)

### Spec and task alignment
- **Proposal alignment**: The implementation matches all 8 acceptance criteria in `proposal.md`.
  - AC1 & AC2: Strict version-1 policy loading via `ai/copilot/retired-settings.json` and `copilotsettings.LoadPolicy` validating dotted paths and rejecting malformed/unknown/overlapping keys. Synthetic test fixtures reproduce `telemetry.disabled` without live credential exposure.
  - AC3 & AC4: `dotf doctor` diagnoses retired settings by file path and dotted key only (redacting secret values). `doctor --fix` removes only retired leaves, prunes newly empty parents, preserves sibling keys, and preserves `.pre-dotf` backup.
  - AC5 & AC6: Optional `deploy.RunOptions.Repair` seam wired into bare/named `dotf deploy` and `configs-deploy` converge. Second run reports 0 changes (idempotent).
  - AC7: Missing settings file is skipped for normal deploy creation; unreadable/malformed/non-object settings fail loud without rewrite or backup creation.
  - AC8: The validation path is self-contained Go inspection and does not invoke or trust `copilot config --list`.
- **Tasks alignment**: All 12 implementation tasks (lines 27–38 in `tasks.md`) are marked `[x]` with corresponding test and code evidence in the diff. Under `## Closing`, lines 45 and 48 (`- [ ] Lint passes` and `- [ ] PR opened referencing this spec folder`) were left unchecked in the contract file prior to PR creation, though PR #3218 is open and CI lint passed.
- **Verification evidence**: Verification commands for all six features in `features.json` were re-executed and pass.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | spec-artifacts | `tasks.md` Closing section leaves `- [ ] Lint passes` and `- [ ] PR opened referencing this spec folder` unchecked, even though PR #3218 is open and CI linters passed. | `specs/BUG-164-copilot-settings-guard/tasks.md:45,48`; PR #3218 open; CI runs passed | UNTESTED | spec (`tasks.md`) / outside contract (`verification.md`) |
| Minor | THEORETICAL | deploy-repair-seam | `deploy.Run` executes `o.Repair` before `DeployWith`. If a future configuration entry carries `render: true` and the renderer returns `ErrRenderIncomplete`, the destination file would already be rewritten on disk while the step outcome reports `kept the installed file`. Currently, only `copilot-settings` (`render: false`) is wired. | `cli/internal/deploy/run.go:88-105` code inspection | `TestRun_OptionalRepairCountsAsAChangeAndHonorsDryRun` (covers seam, not incomplete render) | code (`cli/internal/deploy/run.go`) |
| Minor | THEORETICAL | feature-verification | `features.json` f6 verification command uses `! rg -n ...`. On an environment lacking `rg`, the shell returns exit code 127, which `!` inverts to 0, producing a false-green check if ripgrep is absent. | `specs/BUG-164-copilot-settings-guard/features.json:40` | `TestCheckCopilotSettingsReportsFileAndKeyWithoutValue` (covers negative path in Go) | spec (`features.json`) |
| Minor | THEORETICAL | repo-policy-guard | No test asserts that repository-tracked source `ai/copilot/settings.json` does not contain any key declared in `ai/copilot/retired-settings.json`. If a future commit added a retired setting to source, deploy would re-introduce it after repair. | `cli/internal/copilotsettings/policy_test.go` inspection | UNTESTED | tests (`cli/internal/copilotsettings/policy_test.go`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 8 acceptance criteria verified with comprehensive positive and negative test cases. |
| Verification       | B | Feature verification commands pass and are reproducible; f6 command has theoretical dependency on `rg` presence. |
| Scope              | A | Diff strictly satisfies the bug fix and rollout constraints without deploy schema changes or feature creep. |
| Reliability        | A | Staged atomic writes, idempotent runs, `.pre-dotf` backup preservation, and strict input validation. |
| Maintainability    | A | Clean Go architecture, cyclomatic complexity under 10, functions under 40 lines, table-driven tests. |
| Handoff-readiness  | B | Lesson 559 added and verification documented; two closing tasks left unchecked in contract file prior to PR. |

### Verdict
PASS

### Recommended next steps

Recommendations outside the contract set (for disposition in `verification.md` or follow-up issue):
- **verification.md disposition (outside contract set)**: Record a note in `verification.md` dispositioning the two unchecked closing checklist items in `tasks.md` (`Lint passes` and `PR opened`), noting that PR #3218 was opened and CI checks (`cli-lint`, `lint`, `lint-powershell`, `hygiene`) passed cleanly.
- **code / tests follow-up (outside contract set)**: Consider adding a static unit test in `cli/internal/copilotsettings/policy_test.go` verifying that `ai/copilot/settings.json` contains no keys declared in `ai/copilot/retired-settings.json`.
- **code follow-up (outside contract set)**: In `cli/internal/deploy/run.go`, consider structuring `o.Repair` to run after successful rendering validation to eliminate theoretical drift between `ErrRenderIncomplete` and applied repairs.
- **Archive recommendation**: `dotf spec archive BUG-164-copilot-settings-guard` is **advisable** and ready to execute once PR #3218 is prepared for merge.
