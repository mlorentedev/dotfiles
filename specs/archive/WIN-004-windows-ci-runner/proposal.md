---
id: "WIN-004-windows-ci-runner"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# WIN-004-windows-ci-runner

> **Naming**: file lives at `<repo>/specs/WIN-004-windows-ci-runner/proposal.md`. `WIN-004-windows-ci-runner` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: Add `windows-latest` GH Actions job that executes `setup-windows.ps1` + a PowerShell bats-equivalent suite. CI today runs `ubuntu-latest` only, leaving 1478 LOC of Windows setup with zero execution coverage. Effort: M. Anti-scope: do NOT bundle TEST-001 or POLISH-003. -->

`setup-windows.ps1` is 1478 LOC of Windows-specific deployment logic. CI runs on `ubuntu-latest` only and exercises Windows code paths through PSScriptAnalyzer (static analysis under Linux pwsh) plus a handful of bats tests that mock Windows behavior. Every Windows-only bug shipped since SDD-002 — BUG-005 (PS5.1 re-exec), BUG-020 (Split-Path parameter set), BUG-021 (gitconfig hash check), BUG-022 (profile heal) — escaped CI and surfaced only on real user runs. This is exactly the gap Phase 2.6 (Linux idempotence CI) closes on one side and leaves wide open on the other.

## What

New CI job `test-windows` on `windows-latest` that (1) checks out the repo, (2) executes `setup-windows.ps1` end-to-end, (3) runs `healthcheck.ps1` to verify deployment integrity, (4) runs the Pester suites (`tests/*.Tests.ps1` — `sdd-009-deploy-time-secrets.Tests.ps1` was written for this runner and never executed before), (5) executes the PowerShell-flavored bats subset (`powershell-profile.bats`, `profile-heal-ps1.bats`, `setup-windows.bats`, `init-project-ps1.bats`, `knowledge-crystallize-ps1.bats`, `obs-cli-ps1.bats`, `healthcheck-ps1.bats`). Job becomes required-to-merge in branch protection for `main`.

**CI sandbox** (so both scripts exit 0 on a clean runner): a throwaway age identity is generated and `sensitive/nan.api-key.secret.age` re-encrypted with it in the checkout, so the SDD-009 `{env:NAN_API_KEY}` substitution path executes for real; a minimal vault tree (`00_meta/ 10_projects/ 40_resources/ .obsidian/types.json`) plus an `obsidian` CLI stub satisfy healthcheck section 7; `ANTIGRAVITY_ENDPOINT`/`CLOUDCODE_URL`/`GEMINI_DIR`/`DOTFILES_DIR` are set explicitly (profile not loaded in CI).

**Enabling fix**: both healthchecks' BUG-015 claude-mem hook probe FAILed unconditionally when Claude Code never ran; it is now gated on `installed_plugins.json` (same record as the BUG-014 check) and SKIPs on a clean machine — semantically correct beyond CI.

## Out of scope

- **Bats coverage gaps** — TEST-001 owns adding tests for `claude-mem-heal.{sh,ps1}`, `vault-maintenance-weekly.{sh,ps1}`, `backup-secrets-to-usb.sh`. This PR runs whatever tests already exist; expanding the suite is downstream.
- **Expanded PSScriptAnalyzer coverage** — POLISH-003 owns globbing `scripts/*.ps1`. Two distinct atomic PRs.
- **macOS runner** — DOCS-001 decision pending; if macOS is dropped, no runner needed.
- **Windows idempotence stress test** — mirrors Linux Phase 2.6 and is a separate WIN-XXX.

## Risks / open questions

All resolved at implementation time:

- **R1 (bats on Windows)** → RESOLVED: curl-tarball + `bash install.sh "$HOME/.local"` under Git Bash, pinned to `versions.conf` `BATS_VERSION` (same source as the Linux `test` job; no choco/npm version dependency).
- **R2 (PS 5.1 re-exec)** → RESOLVED: the setup step deliberately uses `shell: powershell` (5.1) so the BUG-005 re-exec path is what CI executes.
- **R3 (git/gh prereqs)** → RESOLVED: setup deploys the repo `.gitconfig` when `~/.gitconfig` is absent (runner case); no step in the job's path calls `gh` against the API; healthcheck env vars are set explicitly in the workflow.
- **R4 (runner cost)** → RESOLVED: `if: github.event_name == 'pull_request'` + `timeout-minutes: 30`. Wall-time measured on the first PR run (AC6).
- **NEW — winget availability on windows-latest** is historically flaky: setup tolerates its absence (skips tool installs with a warning), and a post-setup fallback step installs `jq`/`eza`/`zoxide` via choco only if missing, so healthcheck section 1 stays deterministic either way.

## Acceptance criteria

- [ ] `.github/workflows/ci.yml` includes a `test-windows` job on `windows-latest`.
- [ ] Job runs `setup-windows.ps1` end-to-end and exits 0.
- [ ] Job runs `healthcheck.ps1` and exits 0.
- [ ] Job runs the named PowerShell bats subset; all tests pass.
- [ ] Job is added to branch protection as required for `main`.
- [ ] Total CI wall-time increase ≤ 7 minutes (measure before merge).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § "Session 2026-05-27 — fresh-eyes audit" → WIN-004.
- Existing Linux CI: `.github/workflows/ci.yml` lines 69-102 (`test` job).
- Related: Phase 2.6 (Linux idempotence CI) — Windows mirror tracked here. POLISH-003 (PSScriptAnalyzer breadth) is the static-analysis counterpart.

<!-- archived 2026-06-10 — PR: https://github.com/mlorentedev/dotfiles/pull/325 -->
