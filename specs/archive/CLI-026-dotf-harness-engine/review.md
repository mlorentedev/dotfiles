---
spec: "CLI-026-dotf-harness-engine"
verdict: "PASS"
reviewed_sha: "738ac8e527268166b4fde6851228d1f82e2b5627"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-03"
---

## Adversarial review

**Scope**: CLI-026-dotf-harness-engine
**Sources**: `specs/CLI-026-dotf-harness-engine/{proposal,tasks,verification}.md`, `git diff c575d38b8aef249db1390e097a1fd3a936898761...HEAD`

### Spec and task alignment

The implementation comprehensively fulfills the goal of AUDIT-007 Phase B / PR10: superseding the bash engine `scripts/compile-harness.sh` and its PowerShell twin in `setup-windows.ps1` with a unified, cross-OS Go harness engine under `dotf harness`.

- **AC1 (`dotf harness skills`)**: Renders committed skill records per kind (`skill`, `command`, `prompt`) with exact target matching, auxiliary file synchronization, and stale output pruning. Verified by unit tests (`TestRenderSkillRecord*`, `TestDeploySkills*`) and historically by `tests/harness-skills-parity.bats` during migration.
- **AC2 (Cross-OS byte parity & CRLF)**: Normalizes CRLF inputs to LF so Windows and POSIX checkouts render identical byte streams and hashes. Verified by `TestRenderSkillRecord_CRLFRendersTheLFBytes`, `TestRenderAgentRecord_CRLFRendersTheLFBytes`, `TestCheckDrift_ACRLFRecordValidatesAsLF`, and `TestCompareWithVault_ACRLFVaultIsNotAnEdit`.
- **AC3 (Copilot skill catalog)**: Injects the alphabetical skill catalog into the designated generated region. Verified by `TestDeploySkills_WritesTheCatalogIntoItsRegion`, `TestReplaceCatalogRegion_RefusesBadMarkers`, and `TestCatalogBullets_ByteOrder`.
- **AC4 (`agents`, `doctrine`, `deploy`)**: Subcommands `dotf harness agents` (model/capability resolution), `dotf harness doctrine` (compact ASCII folded payload with character caps), and `dotf harness deploy` compose all surfaces with proper ordering (instructions -> skills -> agents -> presence -> doctrine). Verified by `TestDeployAgents_*`, `TestCompactDoctrine_*`, `TestFoldToASCII`, `TestInjectDoctrine`, `TestDeployDoctrine_*`, and `TestHarnessDeploy_*`.
- **AC5 (`dotf harness check`)**: Offline drift detection verified green via `dotf harness check` and integrated into `dotf doctor` across all OSes (retiring Windows SKIP from #804, verified via `TestCheckRecordDrift`).
- **AC6 (`--against-vault`)**: Flags committed records lagging or drifting from their vault sources. Verified via `TestHarnessCheck_AgainstVaultFlagsAnUnrefreshedVaultEdit`, `TestCompareWithVault_AnUnrefreshedVaultEditIsStale`, and live execution.
- **AC7 (`dotf harness refresh`)**: Complete Go implementation of the vault refresh pipeline with pre-validation of schemas, triggers, and markers before any file modification. Verified via `TestRefresh*`, `cli/internal/harness/refresh.go`, and `cli/internal/cmd/harness_refresh.go`.
- **AC8 (Caller rewire)**: `setup-linux.sh`, `setup-windows.ps1`, and `converge` (`records-harness`) updated to invoke `dotf harness deploy`. Obsolete PowerShell functions (`Deploy-SkillRecord` and helpers) removed. Verified via `tests/setup-fails-loudly.bats`, `tests/setup-windows.bats`, and `TestRecordsHarness_TheGoDeployConvergesTheRealCheckout`.
- **AC9 (Deletion of legacy engine)**: `scripts/compile-harness.sh`, `tests/compile-harness*.bats`, and parity oracles deleted in slice 7 (`79abe7e5`). Guard-grep across codebase confirms zero active invocations remain.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | vault-sync | `tests/harness-doctrine-records.bats` fails against live vault because `harness/skills/autopilot` differs from its vault source | `bats tests/harness-doctrine-records.bats` line 48 reports `[STALE] harness/skills/autopilot differs from its vault source` due to commit `f6fd04e5` on main touching autopilot. Tracked under #2913; demonstrates AC6 detection works | `tests/harness-doctrine-records.bats` | repo / vault (tracked in #2913) |
| Minor | REAL | platform-acceptance | Windows live acceptance executed in clean `dotf vm windows` guest (Windows 11 Arm64) rather than physical work-lab machine | Disclosed in `tasks.md` line 52 and `verification.md` lines 69-70; verified from-zero convergence (`changed=0` on second plan) under release 0.79.0 | `dotf converge --only records-mirror,records-harness` in Windows guest | spec (`verification.md` already dispositions) |
| Minor | REAL | spec-consistency | `proposal.md` lines 72-80 left AC1-AC5 and AC7-AC9 as `[ ]` while AC6 is `[x]`, despite all ACs being completed and verified | Text inspection of `specs/CLI-026-dotf-harness-engine/proposal.md` vs `tasks.md` | Inspection | spec (`proposal.md` is in contract set; do not edit post-review) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All AC1-AC9 criteria implemented, tested, and verified; negative paths and edge cases (CRLF, invalid markers, line caps) fully guarded |
| Verification       | A | Comprehensive evidence across table-driven unit tests, bats suites, mutation checks, and clean-guest Windows VM acceptance |
| Scope              | A | Perfectly adheres to the planned phased port and strangler-fig pattern, retiring legacy twins without scope creep |
| Reliability        | A | Refresh and deploy operations use strict pre-flight validation and transactional semantics to ensure idempotent convergence |
| Maintainability    | A | Idiomatic Go code, 0 golangci-lint issues, gocyclo enforced, clean package boundaries and clear file decomposition |
| Handoff-readiness  | A | Thorough `verification.md`, captured lessons (504, 505, 537), and clean transition to archive readiness |

### Verdict
PASS

### Recommended next steps

The following recommendations are outside the contract set (`verification.md` and repo tracking):
1. **Tracked vault refresh (#2913)**: Synchronize `harness/skills/autopilot` with upstream vault sources on main when upstream updates land.
2. **Archive Spec**: Run `dotf spec archive CLI-026-dotf-harness-engine` to complete the SDD lifecycle and move the spec folder to `specs/archive/CLI-026-dotf-harness-engine/`.
