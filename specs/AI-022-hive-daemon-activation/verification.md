---
tags: [spec, verification]
created: "2026-06-03"
---

# Verification - AI-022-hive-daemon-activation

## Evidence

- [x] `mcp-servers.json` hive args = `hive client` -> `jq -e '.servers[]|select(.name=="hive").args=="hive client"' mcp-servers.json`
- [x] Linux migration + service-install block present -> `grep -q "Activating hive daemon" setup-linux.sh`
- [x] Windows mirror block present -> `grep -q "Activating hive daemon" setup-windows.ps1`
- [x] Version gate (no `hive service` probe that could hang an old hive) -> block uses `uv tool list` + `sort -V` (linux) / `[version]` (windows), asserted by `grep -q "sort -V" setup-linux.sh`
- [x] `setup-linux.sh` syntax + lint -> `bash -n setup-linux.sh` clean; `shellcheck` adds no findings in lines 815-845
- [ ] (post-merge, maintainer) Migration of a real `uvx hive-vault` machine -> observe `hive client` entry + daemon active after setup

## Test status

- `bash -n setup-linux.sh` -> OK
- `shellcheck -f gcc setup-linux.sh` -> no findings within the added block (815-845); pre-existing findings elsewhere untouched
- `jq -e` on `mcp-servers.json` -> hive args == `hive client`
- pwsh parse of `setup-windows.ps1` -> not run here (pwsh unavailable on this host); validated by eyeball + the Windows CI/manual run. Mirrors the existing `Backup-AndRestoreClaudeJson` + `$LASTEXITCODE` pattern already in the file.
- Manual smoke (the hive side, one machine): `uv tool upgrade hive-vault` -> 1.32.0; `hive service install` -> systemd unit active + `/health` ready; `~/.claude.json` flipped to `hive client` (documented in hive `docs/runbooks/daemon-activation.md`).
- No regressions: the activation block is appended after the MCP loop; no existing logic changed.

## Decisions made during implementation

- **Version-gate, not subcommand-probe.** An old hive (< 1.32.0) routes `hive service`
  to the blocking stdio server (it reads stdin and hangs setup). So the block gates on
  the installed package version (`uv tool list` parse + `sort -V` / `[version]`), never by
  running `hive service --help`.
- **Migrate by remove-before-loop, not a hardcoded re-add.** The repo enforces "no hardcoded
  `claude mcp add <known-server>`" (tests/setup-*.bats: all MCP servers come from
  mcp-servers.json) -- the first cut hardcoded `claude mcp add … hive client` and tripped it.
  Fixed by REMOVING a stale `uvx hive-vault` entry *before* the registration loop, so the loop
  re-adds the current definition (`hive client`) from the SSOT. The loop still owns every add;
  the migration is remove-only. This also generalises (any server whose definition changes can
  be force-removed pre-loop to get re-added). The service install stays a separate post-loop
  step that touches no MCP state.
- **Non-fatal everywhere.** Missing systemd `--user` / Task Scheduler, an upgrade lag, or any
  failure downgrades to a warning and leaves the stdio entry — the in-process `hive client`
  fallback means a missing daemon degrades, never breaks a session.

## Promotion candidates

- [ ] Lesson for `90-lessons.md`? no — the reusable lesson (exit-code↔restart-policy, version-gate-not-probe) is captured in the hive repo (`docs/lessons.md` + `90-lessons.md`).
- [ ] ADR-worthy? no — ADR-011 in the hive repo already owns the daemon-model decision.
- [ ] New pattern? no — single-project rollout mechanics.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/AI-022-hive-daemon-activation/`
- [ ] hive #176 ticked / closed with the PR link after multi-machine validation
