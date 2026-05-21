---
id: "WIN-001-healthcheck-ps1"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-21"
tags: [spec, proposal, windows, healthcheck, parity]
template_version: "1.0"
---

# WIN-001-healthcheck-ps1

> PowerShell port of `scripts/healthcheck.sh` (398 LOC, 12 sections) so a Windows install can be validated post-setup with the same surface the Linux side already has via `hc`. Spec sibling of `BUG-006`/`BUG-008`/`BUG-009`/`BUG-010` (load-secrets cross-OS completeness): every Linux post-setup verification tool needs a Windows equivalent or the cross-OS contract is fictional.

## Why

<!-- from 11-tasks.md: PowerShell port of `healthcheck.sh` (10 sections, Windows native). Pester tests. Wired into `setup-windows.ps1` as final step. Independent — start here. -->

Today a Windows setup ends with `doctor.ps1` (env-contract.json drift only — env vars, PATH entries, required + optional binaries). `healthcheck.sh` runs **12 sections covering ~50 distinct assertions** that doctor does NOT — versioned tool paths (`JAVA_HOME/bin/java` actually exists?), symlinks/junctions (`~/.zshrc` points where?), knowledge vault structure (`.obsidian/types.json` present?), secrets integrity (every `env-mapping.conf` entry has a `.age` sibling, no orphans?), OpenCode + Ghostty config deployment, repo↔deploy-dir drift. **None of this is verified on Windows today**, so any deploy regression that doctor doesn't catch (which is most of them) ships silent until manual discovery.

The same drift-class incident BUG-002 + BUG-003 exploit lives here too: every assertion in `healthcheck.sh` only protects Linux; the Windows side runs blind. This spec closes that gap.

## What

Three observable behavior changes after this PR:

1. **New `scripts/healthcheck.ps1`** (~410–450 LOC, mimicking the doctor.sh→doctor.ps1 ratio of 1.07) implementing **12 sections numbered identically to `healthcheck.sh`** so bats parity asserts work cross-OS via simple `grep "section '\X/12'"`. Linux-only sections (tmux §9, ghostty §11 pre-TERM-001-win, drift §12 pre-diff-check-ps1) emit `SKIP` with explanation rather than failing or being silently absent — this preserves the 1-to-12 numbering and makes the "Linux-only by design" decision visible in output.

2. **Auto-wired into `setup-windows.ps1` as final step** (after the existing doctor invocation in section 8c). Non-fatal: a `FAIL` count > 0 emits a `Write-Warn` but does NOT alter `$LASTEXITCODE` of setup-windows.ps1. **NOTE / follow-up**: Linux `setup-linux.sh` does NOT auto-invoke `healthcheck.sh` today. Auto-wiring on Windows-only is a deliberate **temporary divergence** while user evaluates the UX. A `WIN-001-followup-linux-parity` task is implicit: if the Windows auto-wire is kept, mirror it on Linux for full parity. Flagged in verification.md.

3. **Alias `hc`** added to `powershell/profile.ps1` so `hc` runs `pwsh -NoProfile -File $env:DOTFILES_DIR\scripts\healthcheck.ps1` from any directory. Mirrors the Linux convention.

## Out of scope

- **`diff-check.ps1` port.** Section 12 emits `SKIP: diff-check.ps1 not implemented (queued for separate spec)`. Keeps this PR atomic. Opens follow-up: `REFACTOR-003-diff-check-ps1` (or similar id when scheduled).
- **`tmux` Windows port.** Section 9 emits `SKIP: tmux is Linux-only by design`. tmux is intentionally Linux-only per the backlog. No `wsl tmux` fallback (would mask the boundary).
- **`ghostty` Windows port.** Section 11 emits `SKIP: ghostty Windows port pending (TERM-001 is Linux-only currently)`. Will be filled in by a later spec when Ghostty Windows lands.
- **Pester unit tests.** Backlog mentioned "Pester tests" but the Linux side uses **bats grep-based static asserts** (see `tests/healthcheck.bats` — 100% grep + shellcheck, zero runtime execution of healthcheck.sh itself). Cross-OS parity = bats also for the `.ps1`, following the established pattern from `knowledge-crystallize-ps1.bats` + `obs-cli-ps1.bats` (bats invokes pwsh for PSScriptAnalyzer; the rest stays grep-static). No Pester.
- **Auto-wiring into `setup-linux.sh`.** Deliberate divergence (see What §2). Follow-up if Windows UX validates.
- **Refactor of overlap with `doctor.ps1`.** Sections 1 (core tools), 5 (env vars), 6 (optional binaries) intentionally **re-validate** what doctor already checks (defense in depth + broader tool lists), per design decision recorded below.

## Risks / open questions

**Risk: section overlap with doctor.ps1 (sec 1, 5, 6) creates redundant noise.** Mitigation: healthcheck.ps1 sec 1+5+6 use **broader lists** than doctor.ps1 (e.g. doctor's required binaries is just `git` + `jq`; healthcheck core-tools includes `node`, `npm`, `docker`, `kubectl`, `terraform`, `eza`, `zoxide`, etc.). Re-validation is a feature: setup might deploy a tool, doctor might not require it, but healthcheck verifies it landed where expected. The redundancy is intentional — same design as `healthcheck.sh` vs `doctor.sh`.

**Risk: section numbering "X/12" misleads on Windows where 3 sections always SKIP.** Mitigation: SKIP lines are explicit (`SKIP: tmux - Linux-only by design`), and the summary line at the end reports `N passed, M failed, K skipped`. User sees that K=3 minimum on Windows is normal. Bats asserts cross-OS section numbers identically.

**Risk: auto-wiring into setup-windows.ps1 doubles setup runtime tail.** Mitigation: doctor runs in <1s; healthcheck most expensive section is sec 8 (iterates env-mapping entries) and sec 7 (filesystem checks on vault). Empirically <3s total expected. Non-fatal means a slow / flaky check doesn't break setup.

**Risk: PS5.1 fallback (per BUG-005).** Mitigation: setup-windows.ps1 already re-execs under pwsh 7+ at the top (BUG-005, PR #58). healthcheck.ps1 inherits this — it will only ever run under pwsh 7+. PSScriptAnalyzer asserts no pwsh 7+ APIs that would break under 5.1 if someone invokes the script directly (intentional: a direct invocation under 5.1 should fail loud, not silently degrade).

**Risk: knowledge-vault path drift cross-OS.** Mitigation: `$env:VAULT_DIR` or fallback `$env:USERPROFILE\Projects\knowledge` (mirror of Linux `$HOME/Projects/knowledge`). Same per-OS default as elsewhere in the repo.

**Open question (resolved): does sec 4 "Key Symlinks" port?** Windows uses **file copies** for profile.ps1 + CLAUDE.md etc. (no symlink/junction unless explicit). Decision: rename sec 4 to "Key Files / Junctions" — check existence (not symlink-ness) for the Windows-deployed targets (`$env:USERPROFILE\Documents\PowerShell\profile.ps1`, `$env:USERPROFILE\.claude\CLAUDE.md`, `$env:USERPROFILE\.gemini\GEMINI.md`, `$env:USERPROFILE\.ssh\config`, the `marketplaces\thedotmack` junction from BUG-012). Bats grep stays simple (`grep "section '4/12'"`).

## Acceptance criteria

- [ ] `scripts/healthcheck.ps1` exists, valid PowerShell syntax (passes `pwsh -NoProfile -Command "[scriptblock]::Create((Get-Content scripts\healthcheck.ps1 -Raw))"`)
- [ ] PSScriptAnalyzer clean (Error+Warning) using repo `.PSScriptAnalyzerSettings.psd1`
- [ ] 12 sections present and numbered `1/12` through `12/12` (parity with healthcheck.sh)
- [ ] Sections 9 (tmux), 11 (ghostty), 12 (drift) emit `SKIP` lines with explanation on Windows (Linux-only by design / pending follow-up specs)
- [ ] Has `[CmdletBinding()]`, `Set-StrictMode -Version Latest`, `$ErrorActionPreference = 'Continue'` (mirrors doctor.ps1 conventions)
- [ ] Defines `Write-Pass` / `Write-Fail` / `Write-Skip` / `Write-Section` helpers (mirrors doctor.ps1 helper layout)
- [ ] Counters: `$script:Passed` / `$script:Failed` / `$script:Skipped` updated by helpers; final summary line: `Results: N passed, M failed, K skipped`
- [ ] Exit code policy: `exit 0` if `$script:Failed -eq 0`, else `exit 1`
- [ ] `setup-windows.ps1` invokes `healthcheck.ps1` **after** the doctor section (8c → new 8d), non-fatal (`Write-Warn` on FAIL, no `exit`)
- [ ] `powershell/profile.ps1` exports alias `hc -> pwsh -NoProfile -File $env:DOTFILES_DIR\scripts\healthcheck.ps1`
- [ ] `tests/healthcheck-ps1.bats` parity asserts: 12 sections present, helper functions defined, SKIP lines for sec 9/11/12, exit code policy, PSScriptAnalyzer clean
- [ ] Empirical smoke on this Windows machine: `hc` runs, prints 12 sections, reports K≥3 skipped (the 3 Linux-only), exit code matches the FAIL count
- [ ] No `tmux` / Linux symlink assertions on Windows (would fail trivially)

## References

- Vault: `10_projects/dotfiles/11-tasks.md` "WIN-001-healthcheck-ps1" backlog entry
- Source script: `scripts/healthcheck.sh` (398 LOC, the artifact being ported)
- Sibling ports already shipped: `doctor.sh`→`doctor.ps1`, `claude-mem-heal.sh`→`claude-mem-heal.ps1`, `knowledge-crystallize.sh`→`knowledge-crystallize.ps1`, `obs-cli.sh`→`obs-cli.ps1`
- Test pattern: `tests/knowledge-crystallize-ps1.bats`, `tests/obs-cli-ps1.bats` (bats invokes pwsh for PSScriptAnalyzer; rest is grep-static)
- Doctor sibling: `scripts/doctor.ps1` (env-contract drift only; healthcheck does what doctor does NOT)
- Related: `BUG-005-setup-ps7-reexec` (PR #58) — the pwsh 7+ guarantee healthcheck.ps1 inherits
- Audit context: `[[30-architecture/audit-002-cross-os-duplication]]` flagged healthcheck as a missing cross-OS pair
