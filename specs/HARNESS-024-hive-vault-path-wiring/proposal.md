---
id: "HARNESS-024-hive-vault-path-wiring"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-18"
issue: "mlorentedev/dotfiles#446"
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-024-hive-vault-path-wiring

> Follow-up to CLI-016 (#445) / ADR-025. Wires the two MCP-layer vault-path consumers that ADR-025 build-sequence step 4 missed.

## Why

<!-- from issue #446: wire resolved vault path into setup + Claude Code hive MCP server -->

CLI-016 (#445) activated the ADR-025 cascade (`env → ~/.config/dotfiles/machine.json → env-contract default`) for shells, hooks, the Go CLI, and `dotf env generate`. But two consumers of the vault path resolve it from a **hardcoded old literal**, so the durable fix does not actually repair the Hive of Claude Code (or Antigravity):

- **GAP 1** — setup hardcodes `~/Projects/knowledge` for the agy `hive-vault` MCP registration and the auto-memory junction scan: `setup-windows.ps1:1074` (agy `env.VAULT_PATH`), `setup-windows.ps1:650` and `setup-linux.sh:1214` (junctions). `setup-linux.sh:402/499` were already cascade-aware (`${VAULT_PATH:-…}`) — Windows had drifted out of sh↔ps1 parity.
- **GAP 2** — the Claude Code `hive` MCP server (`mcp-servers.json`, `args: "hive client"`) has **no env block**, so setup never sets the vault path for it. `hive client` is a stateless proxy to the `hive serve` daemon, which reads `HIVE_VAULT_PATH` from its **own process env at start** — and the daemon (systemd `--user` on Linux, Scheduled Task on Windows, installed via `hive service install`) is **not** given any env. So it kept the stale path.

## What

- **GAP 1 — cascade parity in setup.** Make the three hardcoded sites honor `$VAULT_PATH` first (`${VAULT_PATH:-default}` / `if ($env:VAULT_PATH) {…}`), reaching sh↔ps1 parity. No behaviour change where the seam is set; correct resolution where the vault moved.
- **GAP 2 — provision the daemon's environment from the cascade.** The daemon does not source the shell path file, so the vault path must reach it via the OS service-env mechanism:
  - **`dotf env path <KEY>`** — new subcommand printing the cascade-resolved value of one key (wraps `env.ResolvePath`), so setup can resolve `HIVE_VAULT_PATH` without sourcing.
  - **Linux:** setup writes `~/.config/environment.d/10-hive-vault.conf` (read by the systemd `--user` manager for all user services) with the resolved `HIVE_VAULT_PATH`/`VAULT_PATH`.
  - **Windows:** setup persists `HIVE_VAULT_PATH` at **User scope** (Scheduled Tasks inherit the User environment) from the cascade.
  - **`mcp-servers.json`:** the `hive` server keeps **no** env block by design (the client is a stateless proxy; `claude mcp add` has no `--env`); the decision + the daemon-env mechanism are recorded in its `_history`.
- **Docs robustness.** Point the canonical AI-facing vault-root references (`AGENTS.md` ×3, `.github/copilot-instructions.md`, `ai/copilot/copilot-instructions.md`, the PR template) at `$VAULT_PATH` (default `~/Projects/knowledge`, per-machine override via machine.json, ADR-025) so an agent reading them after a move resolves the seam, not a stale literal.

## Out of scope

- **The daemon's own path-resolution hardening** (precedence, fallbacks, validation inside `hive serve`) — that is **hive#246 (HIVE-119)**, in the hive repo. This spec guarantees the env is *present*; hive#246 makes the daemon *robust*.
- **A `claude mcp add --env` wiring** in the setup MCP registration loop — not needed (the daemon, not the client, holds the path) and tracked separately if it ever is.
- **Migrating the PR-template vault-backlog line off the legacy vault `11-tasks.md`** (ADR-018 moved tracking to the bitácora Project) — only the path reference is updated here.

## Risks / open questions

- **environment.d timing** — read by the systemd `--user` manager at login; an already-running daemon picks it up on its next restart (the hive-upgrade timer restarts every 15 min). Setup also `import-environment`s it for immediate effect. Acceptable.
- **Windows User-scope env** — complements (does not contradict) ADR-025: shells get the path from `paths.ps1`; services (which can't source it) get it from the User env. Same cascade source.
- **`dotf env path` on Windows** — `dotf` install is still on the WIN queue, so the Windows daemon block falls back to the `$env:VAULT_PATH` idiom when `dotf` is absent.

## Acceptance criteria

- [x] No live hardcoded `Projects/knowledge` literal remains for vault-path *resolution* in `setup-{linux,windows}`; all read the cascade. Windows reaches sh parity. (Comment/fallback literals are allowed.)
- [x] `dotf env path VAULT_PATH` prints the cascade-resolved value (machine.json override honored); `go build`/`go test ./...` green.
- [x] Linux setup writes `~/.config/environment.d/10-hive-vault.conf`; Windows setup persists `HIVE_VAULT_PATH` at User scope — both from the resolved cascade.
- [x] `mcp-servers.json` `_history` records why the `hive` server has no env block + where the daemon gets its path.
- [x] Canonical AI vault-root references point at `$VAULT_PATH` + machine.json (AGENTS.md, copilot ×2, PR template).
- [x] `bash -n setup-linux.sh` + PowerShell parser on `setup-windows.ps1` clean; affected bats green.

## References

- ADR: `docs/adr/adr-025-cross-machine-path-resolution.md` (the cascade) · parent CLI-016 (#445)
- Cross-repo: `hive#246` (HIVE-119: harden hive-vault path resolution) — the daemon-side robustness
- Precedent: INFRA-003 (the manual `~/.claude.json` hive env that went stale)
- Work-gate: `mlorentedev/dotfiles#446`
