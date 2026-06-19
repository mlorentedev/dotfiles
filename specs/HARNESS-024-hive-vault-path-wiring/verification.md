---
tags: [spec, verification]
created: "2026-06-18"
---

# Verification - HARNESS-024-hive-vault-path-wiring

## Evidence

- [x] **AC1 — GAP 1 parity** → `grep -n "Projects.knowledge" setup-{linux,windows}` shows only comments + fallbacks; the three resolution sites read `$VAULT_PATH` first (`setup-linux.sh:1214`, `setup-windows.ps1:650/1074`), matching the pre-existing `setup-linux.sh:402` idiom.
- [x] **AC2 — dotf env path** → `cli/internal/cmd/env.go` `newEnvPathCmd`; smoke (DOTFILES_DIR=repo, VAULT_PATH unset): `dotf env path VAULT_PATH` → `C:\Users\mlorente\Projects\Workspace\knowledge` (machine.json override honored). `go build`/`go test ./...` green.
- [x] **AC3 — daemon env** → `setup-linux.sh` writes `~/.config/environment.d/10-hive-vault.conf` (+ `import-environment`) after `hive service install`; `setup-windows.ps1` persists `HIVE_VAULT_PATH` at User scope. Both resolve via `dotf env path` with an env-idiom fallback.
- [x] **AC4 — mcp-servers.json** → `_history` records the no-env-block rationale (stateless `hive client`; daemon reads its own env; `claude mcp add` has no `--env`) and the environment.d / User-scope mechanism.
- [x] **AC5 — docs** → `AGENTS.md` (×3), `.github/copilot-instructions.md`, `ai/copilot/copilot-instructions.md`, `.github/pull_request_template.md` now reference `$VAULT_PATH` + `~/.config/dotfiles/machine.json` + ADR-025.
- [x] **AC6 — syntax/tests** → below.

## Test status

- `go build ./...` → 0. `go test ./...` (from `cli/`) → all `ok` (incl. `internal/cmd`, `internal/env`). `gofmt -l` clean.
- `bash -n setup-linux.sh` → OK. PowerShell `Parser::ParseFile(setup-windows.ps1)` → OK.
- `bats tests/{setup-linux,setup-windows,hive-upgrade-timer,env-contract,agents-md}.bats` → all green **except** `setup-linux.sh valid zsh syntax`, which fails **only because `zsh` is not installed on the Windows dev box** (the test runs `zsh -n`). The added constructs (`${VAR:-…}`, `$(…)`, `printf`, `if/then/fi`) are zsh-compatible and mirror the existing `setup-linux.sh:402` line; CI (ubuntu, has zsh) is the authority and validated the unedited file on #447.

## Decisions made during implementation

- **The daemon, not the MCP client, holds the vault path.** `hive client` is a stateless stdio proxy; `hive serve` reads `HIVE_VAULT_PATH` from its own process env at start. So the fix provisions the **service** environment (environment.d / User scope), not the `~/.claude.json` server `env` (which `claude mcp add` cannot set anyway).
- **Service env is complementary to ADR-025, not a contradiction.** Shells get the path from `paths.{sh,ps1}`; services (which never source a shell profile) get it from environment.d / User env. Both derive from the same cascade (machine.json).
- **Daemon-side robustness is hive#246.** This spec guarantees the env var is present and correct; the daemon's own precedence/fallback/validation is the hive repo's concern.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? **yes (at archive)** — "a 'wire all consumers' change must enumerate non-shell consumers too: services (systemd/Scheduled Task) and MCP daemons don't source shell profiles, so a shell-only env mechanism silently misses them."

## Archive checklist

- [ ] PR merged, closes #446.
- [ ] Deploy verified on ≥1 machine: `dotf env path` resolves; environment.d / User env set; `hive serve` reads the new path (coordinate with hive#246).
- [ ] `proposal.md` `status: archived`; folder → `specs/archive/HARNESS-024-hive-vault-path-wiring/`.
- [ ] Bitácora #446 → Done.
