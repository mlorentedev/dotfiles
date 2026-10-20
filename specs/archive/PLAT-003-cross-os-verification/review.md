---
spec: "PLAT-003-cross-os-verification"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "ec7d6c8eeeabb485ad7983cc29a9a4c07a04e736"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-10-19"
---

## Adversarial review

**Scope**: `PLAT-003-cross-os-verification`
**Sources**: `specs/PLAT-003-cross-os-verification/proposal.md`, `tasks.md`, `verification.md`, `features.json`, and `git diff ab176f3eb50c551ccd82e5e10e74df0df6de222a...HEAD`

### Spec and task alignment

The implementation spans three merged pull requests and one refinement PR:
- **PR 1 (Tier 1)**: Commit `452d35bc` adds the `powershell` catalog entry in `packages.json`, `# pwsh-module:` markers in `versions.conf`, `pwsh-modules` converge reconciler, `dotf doctor` PowerShell section, and `dotf lint powershell` with CI parity checks.
- **PR 2 (Tier 3a)**: Commit `4c651efc` adds `dotf vm linux` driving Lima with an embedded, isolated Ubuntu 24.04 template (`linux.yaml`), verifying a from-zero clone and double converge.
- **PR 3 (Tier 3b)**: Commit `11c3c8cd` and `a436d19c` implement QEMU Windows 11 Arm64 VM management (`dotf vm windows build` and `dotf vm windows run` / `stop`), guest communication over QGA / QMP, unattended installation via embedded `autounattend.xml`, clean-base snapshots, and local source build testing (`--dotf`).
- **Tier 2 (ARM CI runners)**: Declared explicitly deferred in `proposal.md` with no code in this change.

All 10 acceptance criteria defined in `proposal.md` are implemented and supported by automated tests in `cli/internal/pwsh/`, `cli/internal/vm/`, `cli/internal/converge/`, `cli/internal/doctor/`, and `cli/internal/cmd/`. Live verification runs for macOS, Lima Linux, and Windows 11 Arm64 QEMU are recorded with timestamps, commands, and outputs in `verification.md`. Six knowledge lessons (466, 467, 473, 481, 484, 485) and ADR-053 are written and verified on disk.

Minor misalignments identified:
- `features.json` lists only features `f1` through `f8` (AC1 through AC8). AC9 (`dotf vm windows` clean-base revert, archive copy, install.ps1, second converge idempotence) and AC10 (`qemu` tools plan and apply) are omitted from `features.json`.
- `tasks.md` closing tasks lines 53-54 (`verification.md complete for every tier`, `Independent adversarial review before archive`) remain unticked `[ ]`.
- `verification.md` line 39 cites `test/` instead of `tests/` for the bats suite commands.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | vm/qga | `QGA.Run` executes an unbounded polling loop without timeout: in `cli/internal/vm/qga.go:138-154`, `Run` polls `guest-exec-status` until `st.Exited` is true. If a guest process hangs indefinitely (e.g. during `waitLogon` CIM/WMI queries or service queries), `q.Run` never returns and blocks the host thread indefinitely, bypassing the outer deadline. | Inspection of `cli/internal/vm/qga.go:138-154` vs `waitLogon` in `run.go:187-200` | UNTESTED | code (`cli/internal/vm/qga.go`) |
| Minor | REAL | spec-contract | `features.json` omits features for AC9 and AC10: `proposal.md` defines 10 acceptance criteria (AC1–AC10), but `features.json` contains entries only up to `PLAT-003-cross-os-verification-f8`. AC9 and AC10 have no entries in the machine-readable features manifest. | `specs/PLAT-003-cross-os-verification/features.json:1-58` vs `proposal.md:63-67` | UNTESTED | spec (`features.json`, follow-up ticket) |
| Minor | REAL | verification-docs | Path typo in documented bats commands: `verification.md:39` references `test/ci-pester-pin.bats ...` instead of `tests/...`, which exits 1 with `No such file or directory` when executed directly from the repo root. | Executed `bats test/ci-pester-pin.bats ...` -> failed; `bats tests/...` -> 98 ok | UNTESTED | spec (`verification.md`) |
| Minor | THEORETICAL | cmd/lint | `DOTF_PWSH_LINT_FILES` environment block payload size: `runLintPowerShell` in `cli/internal/cmd/lint.go:79` passes all file paths in a single environment variable. On Windows, total environment blocks are capped at 32,767 characters. Current repo has 28 PowerShell files (~1 KB), but large file lists or deeply nested paths could exceed the Windows environment limit. | `cli/internal/cmd/lint.go:79` | `TestLintFiles_SeesWhatACommitWouldShip` | code (`cli/internal/cmd/lint.go`) |
| Minor | REAL | vm/ref-resolution | Asymmetric ref resolution between Linux and Windows harnesses: `dotf vm linux` requires `--ref` to be pushed to remote GitHub (`git fetch origin <ref>`), whereas `dotf vm windows` packages local committed git objects via `git archive` and copies them via QGA. Unpushed local commits fail on Linux with `fatal: couldn't find remote ref`. | Documented in `verification.md:21` and `cli/internal/vm/vm.go:115` | `TestLinuxTemplate_SharesNothingWithTheHost` | docs / follow-up |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All 10 ACs implemented and verified; negative paths covered; minor QGA polling timeout gap. |
| Verification       | A | Comprehensive evidence with exact commands, outputs, and live VM runs across Linux and Windows. |
| Scope              | A | Perfectly matches the proposal; delivered Tiers 1, 3a, 3b while cleanly deferring Tier 2. |
| Reliability        | B | Clean teardown handlers and idempotent double-converges verified; unbounded QGA process wait noted. |
| Maintainability    | A | Strict types, table-driven tests, 0 golangci-lint issues, and cyclomatic complexity strictly held. |
| Handoff-readiness  | B | 6 lessons and ADR-053 committed; minor features.json sync gap to be ticketed. |

### Verdict
PASS WITH GAPS

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is closed. Do **not** edit contract files in this session, as editing them would invalidate this review's contract digests and stale the verdict.

The implementer should disposition the findings in `verification.md` (which is outside the contract set and excluded from staleness checks):

1. **In `verification.md` (non-contract set, safe to edit)**:
   - Fix the command path typo on line 39 (`test/` -> `tests/`).
   - Add a `## Review triage` section recording dispositions for each finding:
     - Finding 1 (features.json f9/f10): Ticket for follow-up spec maintenance.
     - Finding 2 (verification.md test path): Applied in verification.md.
     - Finding 3 (QGA.Run timeout): Ticket as resilience enhancement for `dotf vm windows`.
     - Finding 4 (DOTF_PWSH_LINT_FILES size): Decline / ticket (acceptable for current repository scale).
     - Finding 5 (ref resolution asymmetry): Decline with rationale (by design due to Lima plain isolation).

2. **Follow-up work (post-archive)**:
   - Archive the spec using `dotf spec archive PLAT-003-cross-os-verification`.
