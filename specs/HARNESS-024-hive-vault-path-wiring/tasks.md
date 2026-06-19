---
tags: [spec, tasks]
created: "2026-06-18"
---

# Tasks - HARNESS-024-hive-vault-path-wiring

> One PR, stacked on CLI-016 (#445). Closes #446. Follows ADR-025; daemon-side robustness is hive#246.

## Setup

- [x] Branch `feat/hive-vault-path-wiring` off the CLI-016 branch (needs the `env` package + setup `dotf env generate` wiring).

## GAP 1 — setup cascade parity

- [x] `setup-linux.sh:1214` `VAULT_ROOT` → `${VAULT_PATH:-$HOME/Projects/knowledge}` (auto-memory junctions).
- [x] `setup-windows.ps1:650` `$VaultRoot` → `if ($env:VAULT_PATH) {…} else {…}` (junctions).
- [x] `setup-windows.ps1:1074` `$vaultPath` → cascade-aware (agy `hive-vault` `env.VAULT_PATH`), matching `setup-linux.sh:402`.

## GAP 2 — daemon environment provisioning

- [x] `dotf env path <KEY>` subcommand (`cli/internal/cmd/env.go`) wrapping `env.ResolvePath`.
- [x] `setup-linux.sh`: after `hive service install`, write `~/.config/environment.d/10-hive-vault.conf` from the cascade (`dotf env path` with `${…:-…}` fallback) + `systemctl --user import-environment`.
- [x] `setup-windows.ps1`: after `hive service install`, persist `HIVE_VAULT_PATH` at User scope from the cascade (`dotf env path` when available, else the `$env:VAULT_PATH` idiom).
- [x] `mcp-servers.json` `_history`: record the no-env-block decision + the daemon-env mechanism.

## Docs robustness

- [x] `AGENTS.md` ×3 (CORE PRINCIPLE vault location, patterns path, spec SKILL path) → `$VAULT_PATH` + machine.json + ADR-025.
- [x] `.github/copilot-instructions.md` + `ai/copilot/copilot-instructions.md` vault-root → seam.
- [x] `.github/pull_request_template.md` vault-backlog path → `$VAULT_PATH`.

## Closing

- [x] `go build` / `go test ./...` green; `dotf env path VAULT_PATH` smoke (machine.json honored).
- [x] `bash -n setup-linux.sh` + PowerShell parser on `setup-windows.ps1` clean. (Local `zsh -n` test skipped — zsh absent on the dev box; CI ubuntu validates.)
- [x] `verification.md` filled.
- [ ] PR opened referencing this spec folder; closes #446.

## Machine-readable features

`features.json` is emitted alongside; the harness sets `"state": "passing"` after a green `verification` command.
