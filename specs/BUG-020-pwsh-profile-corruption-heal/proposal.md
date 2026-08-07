---
id: "BUG-020-pwsh-profile-corruption-heal"
type: spec
status: implementing
created: "2026-05-27"
tags: [spec, proposal, win, profile, secrets]
template_version: "1.0"
---

# BUG-020-pwsh-profile-corruption-heal

Recovery + prevention pair: BUG-020 (heal script) and BUG-021 (setup-windows fail-fast preflight).
Shipped together in PR [#120](https://github.com/mlorentedev/dotfiles/pull/120) because the preflight error message points at the heal script — split PRs would yield a broken pointer.

## Why

A real-machine session-start error surfaced on Windows 2026-05-26: every PowerShell session emitted `ParserError: ... Missing closing ')' after expression in 'if' statement` at line 880 of the deployed profile, plus a 14-second profile load. Investigation: the deployed profile (`%USERPROFILE%\Documents\PowerShell\Microsoft.PowerShell_profile.ps1`) had grown to **26 MB / 689,366 lines / 3,186 START markers / 5,465 parse errors**. Root cause: `setup-windows.ps1:993-1015` ran `$existingContent -replace $pattern, $newSection` and `Set-Content`. On large/corrupted profiles, `-replace` aborts with `OperationStopped: Insufficient memory`, but the script printed `[SUCCESS] Updated dotfiles section in PowerShell profile` **regardless** — no `try/catch`, no `-ErrorAction Stop`, no `$?` evaluation. That silent failure compounded across runs, interacting with the thedotmack `claude-mem` plugin's habit of inserting `function claude-mem { ... }` outside the dotfiles markers, until the file crossed an unrecoverable threshold.

## What

After this PR:

- `setup-windows.ps1`'s profile-section block (a) preflights known corruption signals — size > 1 MB OR start/end marker count > 2 — and aborts with a one-line pointer to `scripts\profile-heal.ps1` instead of attempting `-replace`; (b) wraps the read/modify/write block in `try/catch` with `-ErrorAction Stop` on all I/O cmdlets; (c) exits non-zero on any failure with a recovery hint. The unconditional `[SUCCESS]` print is gone.
- `scripts/profile-heal.ps1` (new, ~165 LOC) reconstructs a corrupted profile from the SSOT `powershell/profile.ps1`, after backing up the existing file to `<profile>.corrupted-yyyyMMdd-HHmmss.bak`. Idempotent: silent no-op on healthy profiles, prints one line per heal action.
- `doctor.ps1 -Fix` invokes `profile-heal.ps1` in the same loop as `claude-mem-heal.ps1`.
- `setup-windows.ps1` deploys `profile-heal.ps1` alongside other scripts.

## Out of scope

- Preserving user customizations OUTSIDE the dotfiles markers when healing. The current heal script wipes the file and rebuilds it; any user-injected content (e.g., plugin lines) gets backed up but not re-merged. Acceptable because (a) the backup is intact, (b) most external injections (claude-mem plugin) re-add themselves on next session, (c) attempting selective preservation risks re-importing the corruption.
- Rewriting the marker-based section approach on Windows. Preserving user customizations OUTSIDE markers is the explicit design intent of the section approach; the corruption issue is orthogonal (silent failure, not algorithm fault).
- Linux side changes. `setup-linux.sh` deploys `.bashrc`/`.zshrc` via `deploy_file` (clean-copy under `set -euo pipefail`); no accumulation path exists. The asymmetry is intentional and documented in `tests/profile-heal-ps1.bats`.
- SDD-009 (`opencode.jsonc` deploy-time secret substitution): separate ticket, separate PR.

## Risks / open questions

- **bats-on-Windows gather-tests deadlock** on `tests/setup-windows.bats` (103 tests). Reproducible locally; CI Linux unaffected. Mitigation: validate assertions via direct grep + `[System.Management.Automation.Language.Parser]::ParseFile`. Follow-up issue if needed.
- **Threshold drift**: the size/marker thresholds (1 MB, > 2 markers) live in two places — `setup-windows.ps1` preflight and `profile-heal.ps1` corruption detection. They must stay in sync. Current bats regex accepts both naming conventions (`markerCount` vs `StartMarkers`). Future refactor could extract to a shared constant in `scripts/utils.ps1` (not in this PR).
- **Plugin re-injection cycle**: thedotmack `claude-mem` plugin re-adds `function claude-mem { ... }` outside markers on next session. Each re-add is harmless (1 line) but accumulates if repeated. Future hardening: heal script could detect and dedupe known plugin injections.

## Acceptance criteria

- [x] `scripts/profile-heal.ps1` exists and reconstructs the profile from `powershell/profile.ps1` SSOT, backing up the existing file with a timestamped suffix.
- [x] `doctor.ps1 -Fix` invokes `profile-heal.ps1` when the script is present.
- [x] `setup-windows.ps1` deploys `profile-heal.ps1` to `$ScriptsDir`.
- [x] `setup-windows.ps1` preflights the profile (size > 1 MB OR marker count > 2) and aborts with a pointer to `profile-heal.ps1` instead of attempting a potentially-failing `-replace`.
- [x] `setup-windows.ps1` profile-section is wrapped in `try/catch` with `-ErrorAction Stop` and exits non-zero on failure.
- [x] bats coverage: 6 tests in `tests/setup-windows.bats` (BUG-021 fail-fast contract) + 16 tests in `tests/profile-heal-ps1.bats` (BUG-020 recovery contract + setup deploy + doctor integration + Linux asymmetry).
- [x] Smoke test: run `profile-heal.ps1` against an actual corrupted profile (26 MB) → reduce to healthy size, zero parser errors, backup preserved for rollback.
- [x] Cross-OS parity: `setup-linux.sh` retains `set -euo pipefail` + `deploy_file` for `.bashrc`/`.zshrc` (asserted by `tests/profile-heal-ps1.bats`'s parity test).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (BUG-020 + BUG-021 entries)
- GH issues: [#112](https://github.com/mlorentedev/dotfiles/issues/112) (BUG-020), [#113](https://github.com/mlorentedev/dotfiles/issues/113) (BUG-021)
- PR: [#120](https://github.com/mlorentedev/dotfiles/pull/120)
- Companion script: `scripts/claude-mem-heal.ps1` (similar heal pattern, prior art for the doctor.ps1 invocation loop)
- Linux equivalent (clean-copy): `scripts/utils.sh` `deploy_file()` at line 454

## Follow-up (BUG-022)
