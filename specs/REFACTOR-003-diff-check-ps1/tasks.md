---
tags: [spec, tasks, cross-os-parity, drift-detection]
created: "2026-05-21"
---

# Tasks - REFACTOR-003-diff-check-ps1

## Setup

- [x] Branch `feat/REFACTOR-003-diff-check-ps1` (off main).
- [x] Spec scaffolded via `init-spec.ps1` (vault gate satisfied).
- [x] Read `scripts/diff-check.sh` (117 lines) to confirm allowlist + exit semantics + verbose flag behaviour.

## Implementation (TDD order)

### Tests first

- [ ] `tests/setup-windows.bats`: assertion that `setup-windows.ps1` deploys `diff-check.ps1` to `$ScriptsDir` (mirrors doctor.ps1 deploy line).
- [ ] `tests/setup-windows.bats`: assertion that `powershell/profile.ps1` declares `dch` function/alias pointing at `diff-check.ps1`.
- [ ] `tests/healthcheck-ps1.bats`: replace the existing "sec 12 is SKIP" assertion with "sec 12 invokes diff-check.ps1".
- [ ] Run bats — assertions should FAIL (red).

### Implementation

- [ ] `scripts/diff-check.ps1`: port `diff-check.sh` to PowerShell preserving the bash semantics:
  - `-VerboseOutput` and `-Help` switches (PowerShell idiom)
  - Same allowlist regex (`versions.conf|.zshrc|.bashrc|.profile|.gitconfig|tmux.conf` + dir prefixes)
  - `git ls-files` walk; `Get-FileHash SHA256` byte-equal comparison
  - Exit codes: 0/1/2
  - ASCII-only, no em-dashes
- [ ] `setup-windows.ps1`: add `Copy-Item` line for `diff-check.ps1` next to the `doctor.ps1` deploy block (mirror PR #74 healthcheck deploy pattern).
- [ ] `scripts/healthcheck.ps1` sec 12: replace the current SKIP block with an actual invocation of `diff-check.ps1` (non-fatal, surfaces drift count).
- [ ] `powershell/profile.ps1`: declare `dch` function calling `diff-check.ps1` (mirror Linux alias).
- [ ] Run bats — assertions GREEN.

### Lint + cross-check

- [ ] PowerShell AST `[Parser]::ParseFile` clean on all 4 touched `.ps1` files.
- [ ] `Invoke-ScriptAnalyzer -Settings .PSScriptAnalyzerSettings.psd1 -Severity Error,Warning` clean.
- [ ] ASCII-only check (zero non-ASCII chars) on `diff-check.ps1`.
- [ ] Empirical run on user's Windows: deploy `diff-check.ps1` manually, run with no drift expected → exit 0; touch a managed file → exit 1.

## Closing

- [ ] `verification.md` filled with empirical evidence (pre/post-fix healthcheck sec 12 output, dch alias verification).
- [ ] PR opened referencing `specs/REFACTOR-003-diff-check-ps1/`.
- [ ] Post-merge: archive `specs/REFACTOR-003-...` to `specs/archive/`.
- [ ] Post-merge: tick vault `11-tasks.md` REFACTOR-003 entry → ✓ with PR link.
- [ ] Post-merge: append lesson candidate to `90-lessons.md` if any non-obvious decision surfaced (e.g. cross-OS allowlist divergence handling).
