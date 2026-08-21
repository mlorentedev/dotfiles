---
id: "POLISH-003-pwsh-analyzer-coverage"
type: spec
status: archived
created: "2026-05-27"
issue: "mlorentedev/dotfiles#1090"
tags: [spec, proposal]
template_version: "1.0"
---

# POLISH-003-pwsh-analyzer-coverage

> **Naming**: file lives at `<repo>/specs/POLISH-003-pwsh-analyzer-coverage/proposal.md`. `POLISH-003-pwsh-analyzer-coverage` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: Expand PSScriptAnalyzer scan from 5 hardcoded `.ps1` files to all ~22 in `scripts/`. Effort: S, ~20 LOC YAML. Anti-scope: don't tighten severity in same PR — change scope first. -->

`.github/workflows/ci.yml:49-55` runs PSScriptAnalyzer against a hardcoded list of 5 PowerShell files. The repo ships ~22 `.ps1` files in `scripts/` plus `setup-windows.ps1` and `powershell/profile.ps1`, so ~17 PowerShell scripts ship to users with **zero static analysis** — including `claude-mem-heal.ps1` (269 LOC, heaviest bug history), `healthcheck.ps1` (578 LOC), `claude-session-start.ps1` (440 LOC), `doctor.ps1` (236 LOC), and `load-secrets.ps1` (405 LOC). Every Windows bug class shipped (BUG-005, BUG-020, BUG-021) would have been caught earlier under full-tree analysis.

## What

The `lint-powershell` job replaces the hardcoded array with dynamic discovery (`Get-ChildItem -Path scripts -Filter *.ps1`) plus the two root scripts. After this PR, every `.ps1` file in the repo is analyzed by PSScriptAnalyzer on every push and PR.

## Out of scope

- **Severity tightening.** Today's job runs `-Severity Error,Warning`. Adding `Information` (or custom rules) is a separate ticket — scope expansion first, sharpness later.
- **Per-script suppressions.** If the broader scan surfaces noise on legacy scripts, suppress in-line per finding with a 1-line justification rather than gating the PR on a full cleanup.
- **Linux-pwsh-only quirks.** The job already runs on `ubuntu-latest` pwsh; Windows-only syntax issues are a separate WIN-XXX.

## Risks / open questions

- **R1**: Expanded scan will likely surface new warnings on previously-unscanned scripts. Acceptable failure mode: triage pass before merging — fix or suppress with justification. Must merge clean.
- **R2**: Use PowerShell-native `Get-ChildItem` for discovery, not shell glob — keeps behavior deterministic across runner OS.
- **R3**: `.PSScriptAnalyzerSettings.psd1` may be tuned for the prior 5-file list. Audit and adjust if it filters out anything load-bearing for the broader set.

## Acceptance criteria

- [ ] `.github/workflows/ci.yml` `lint-powershell` step uses `Get-ChildItem`-based discovery (no hardcoded file list).
- [ ] CI run against this PR analyzes all 22+ PowerShell files and reports findings per file.
- [ ] PR merges green — zero new warnings, or each new warning has an inline `# PSScriptAnalyzer disable=...` with a 1-line justification.
- [ ] `.PSScriptAnalyzerSettings.psd1` reviewed; any tuning adjustments documented in PR body.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § "Session 2026-05-27 — fresh-eyes audit" → POLISH-003.
- CI workflow: `.github/workflows/ci.yml:36-67` (lint-powershell job).
- Related bugs that would have been caught: BUG-005, BUG-020-SplitPath-LiteralPath-Parent, BUG-021-gitconfig-hash-check.

<!-- archived 2026-08-21 — PR: https://github.com/mlorentedev/dotfiles/pull/1100 -->
