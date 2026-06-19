---
tags: [spec, tasks]
created: "2026-06-18"
---

# Tasks - CLI-016-dotf-env

> TDD order. One PR (single coherent change; the ADR is the upstream "why"). Closes #445.

## Setup

- [x] Branch created from main: `feat/cross-machine-path-resolution`
- [x] ADR-023 accepted (architecture session: state verify → industry audit → constraints → options → decision)
- [x] `proposal.md` complete; acceptance criteria testable

## Contract (SSOT first)

- [x] Add `VAULT_PATH` + `HIVE_VAULT_PATH` to `env-contract.json` with per-OS defaults (no `path_exists` — vault may legitimately be absent; `checkVault` owns existence).
- [x] Update the contract `_comment` to describe the defaults→machine.json cascade.

## Resolver + generator (`cli/internal/env`, TDD)

- [x] Test: cascade resolves override>default, skips no-default vars, expands home, darwin→linux fallback.
- [x] Implement `Resolve` + `defaultFor` + `expand`.
- [x] Test: `Render` (sh) keeps an already-set value via `${VAR:-…}`; (ps1) assigns only `if (-not $env:VAR)`.
- [x] Implement `Render` + `FormatForOS` + `DefaultOutput`.
- [x] Test: `Generate` writes, is idempotent on identical content, and `--check` detects drift.
- [x] Implement `Generate` + loaders (`loadContract`, `loadMachine` — absent machine.json is valid) + `ResolveContractPath` / `MachinePath` / `Home` / `DotfilesDir` + `ResolvePath` (single-key cascade for other Go callers).

## Command wiring

- [x] `cli/internal/cmd/env.go`: `dotf env` parent (`RunE: cmd.Help`) + `generate` subcommand (`--output`, `--check`, `--stdout`).
- [x] `root.AddCommand(newEnvCmd())`.

## Integrations

- [x] `vault.ResolveVault` → `env.ResolvePath("VAULT_PATH")` (drop hardcoded fallback; remove now-unused `path/filepath` import).
- [x] `doctor`: `checkPathFiles` drift check (imports `env`; doctor→env, no cycle) wired into `Run`; `checkVault` `VAULT_DIR`→`VAULT_PATH`.
- [x] `.bashrc` / `.zshrc` / `powershell/profile.ps1`: source generated path file + bootstrap fallback (delete unconditional `DOTFILES_REPO_DIR` export).
- [x] `claude-session-start.{sh,ps1}`: read `$VAULT_PATH` with legacy default fallback.
- [x] `setup-linux.sh` / `setup-windows.ps1`: run `dotf env generate` after deploying the contract; source the result.
- [x] `~/.config/dotfiles/machine.json` for this machine + `machine.json.example` + `.gitignore` entries (`/machine.json`, `/paths.sh`, `/paths.ps1`).

## Closing

- [x] `go build` / `go vet` / `go test ./...` green; `gofmt -l` clean.
- [x] Smoke: `dotf env generate` writes paths.ps1 with machine.json overrides; `--check` → up to date; `dotf doctor --verbose` shows the seams + drift check PASS.
- [x] `verification.md` filled with evidence.
- [ ] PR opened referencing this spec folder; closes #445.

## Machine-readable features

`features.json` is emitted alongside this file; the harness sets `"state": "passing"` after capturing a green `verification` command.
