---
id: "WIN-004-windows-ci-runner"
type: spec
status: draft # draft | implementing | verifying | archived
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

New CI job `test-windows` on `windows-latest` that (1) checks out the repo, (2) executes `setup-windows.ps1` end-to-end, (3) runs `healthcheck.ps1` to verify deployment integrity, (4) executes the PowerShell-flavored bats subset (`powershell-profile.bats`, `profile-heal-ps1.bats`, `setup-windows.bats`, `init-project-ps1.bats`, `knowledge-crystallize-ps1.bats`, `obs-cli-ps1.bats`, `healthcheck-ps1.bats`). Job becomes required-to-merge in branch protection for `main`.

## Out of scope

- **Bats coverage gaps** — TEST-001 owns adding tests for `claude-mem-heal.{sh,ps1}`, `vault-maintenance-weekly.{sh,ps1}`, `backup-secrets-to-usb.sh`. This PR runs whatever tests already exist; expanding the suite is downstream.
- **Expanded PSScriptAnalyzer coverage** — POLISH-003 owns globbing `scripts/*.ps1`. Two distinct atomic PRs.
- **macOS runner** — DOCS-001 decision pending; if macOS is dropped, no runner needed.
- **Windows idempotence stress test** — mirrors Linux Phase 2.6 and is a separate WIN-XXX.

## Risks / open questions

- **R1**: bats on Windows. Use git-bash bats via `choco install bats` or curl-tarball install. Pin the bats version to `versions.conf` `BATS_VERSION` (already used by Linux CI) for parity.
- **R2**: `setup-windows.ps1` line 48 auto-re-execs under pwsh when invoked from PS 5.1. `windows-latest` ships both — verify the re-exec path runs and exits 0 in CI, not just direct-pwsh invocation.
- **R3**: Some Windows-only scripts require `git config --global user.{name,email}` and `gh auth status`. CI runner lacks both. Either set them in a setup step or guard with `if (gh auth status -e SilentlyContinue)` in the scripts.
- **R4**: Runner cost. `windows-latest` minutes are 2x Linux. Mitigate by running only on `pull_request` events targeting `main`, not on every push.

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
