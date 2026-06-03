---
id: "TEST-001-bats-coverage-gaps"
type: spec
status: archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# TEST-001-bats-coverage-gaps

> **Naming**: file lives at `<repo>/specs/TEST-001-bats-coverage-gaps/proposal.md`. `TEST-001-bats-coverage-gaps` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: Add bats tests for `claude-mem-heal.{sh,ps1}` (heaviest bug history — BUG-016/017/018, paused spec at `2026-05-27-claude-mem-heal-consumer-epipe`), `vault-maintenance-weekly.{sh,ps1}`, `backup-secrets-to-usb.sh`. Effort: M, ~150 LOC bats. Anti-scope: only named files. -->

Bats coverage across the repo is ~41 `.bats` files vs. 49 `.sh`/`.ps1` scripts — 1:1 by file count. But the **heaviest-bug-history** scripts have zero coverage. `claude-mem-heal.{sh,ps1}` has 6+ bug fixes (BUG-016, BUG-017, BUG-018, BUG-020 series, plus the paused `2026-05-27-claude-mem-heal-consumer-epipe`) and no bats. `vault-maintenance-weekly.{sh,ps1}` runs unattended on a schedule. `backup-secrets-to-usb.sh` manipulates encrypted artifacts where failure modes are silent. Test debt concentrates exactly where regression risk is highest.

## What

Four new bats files: `tests/claude-mem-heal.bats`, `tests/claude-mem-heal-ps1.bats`, `tests/vault-maintenance-weekly.bats`, `tests/backup-secrets-to-usb.bats`. Each covers: (1) happy path, (2) idempotency (run twice → second run is no-op), (3) the specific failure modes named in prior bug history (EPIPE handling, race conditions, missing files, bad permissions). All run in CI alongside the existing suite.

## Out of scope

- Refactoring the scripts under test.
- Coverage for already-tested scripts.
- Fixing the paused `2026-05-27-claude-mem-heal-consumer-epipe` spec — that's a separate future spec (canonical hooks.json template).
- Coverage for other untested scripts not in the named list (e.g., `nan-*.sh`) — pick those off later if pattern proves valuable.

## Risks / open questions

- **R1**: `claude-mem-heal` manipulates `~/.claude/.mcp.json` and other user-state files. Tests need a sandboxed `$HOME` per test (use `setup()` with `TMP_HOME=$(mktemp -d)` + `export HOME="$TMP_HOME"`).
- **R2**: `backup-secrets-to-usb` expects a USB mount point. Tests mock the mount as a tmpdir.
- **R3**: `vault-maintenance-weekly` calls `obs-cli` / Obsidian CLI. Mock or skip the GUI-bound parts using the existing `OBS_CLI_DRY_RUN` pattern.
- **R4**: PowerShell bats — Windows bats coverage path; rely on the cross-OS bats infrastructure (cf. `powershell-profile.bats`). For now, the `.ps1` test runs in WIN-004's `windows-latest` job; on Linux CI it skips with `[[ "$OSTYPE" == "msys"* ]]` guard.

## Acceptance criteria

- [ ] `tests/claude-mem-heal.bats` exists; covers idempotency + the BUG-017 race-condition class via mocked candidates + the EPIPE class from the paused spec.
- [ ] `tests/claude-mem-heal-ps1.bats` exists; mirrors `.sh` coverage for Windows.
- [ ] `tests/vault-maintenance-weekly.bats` exists; covers `.sh` and `.ps1` happy paths via cross-OS dispatch.
- [ ] `tests/backup-secrets-to-usb.bats` exists; covers happy path + missing-mount + bad-permissions.
- [ ] All new tests pass in CI on `ubuntu-latest` (and `windows-latest` when WIN-004 lands).
- [ ] README "Tested" line updated with new total bats count.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § "Session 2026-05-27 — fresh-eyes audit" → TEST-001.
- Bug history for `claude-mem-heal`: BUG-016, BUG-017, BUG-018, BUG-020, paused spec `2026-05-27-claude-mem-heal-consumer-epipe`.
- Existing per-script bats convention: `tests/profile-heal-ps1.bats`, `tests/healthcheck-ps1.bats`.
