# AGENTS.md (Hermes)

> **SYSTEM META-INSTRUCTION:** Target agent: Hermes Agent (Nous Research), a
> remote ops agent on NaN infrastructure (Debian 13), reachable via Telegram.
>
> **First, read `AGENTS.md` at the repo root** — the canonical SSOT for behaviour
> rules across all agents (Standing Orders, Decision Hierarchy, Neural Hive Loop,
> MCP usage, Operational Rules). This file holds only Hermes-specific notes on top.
>
> If `AGENTS.md` is missing from the current context, default to the canonical
> version at `$DOTFILES_REPO_DIR/AGENTS.md` (resolved via `machine.json` per
> ADR-025; falls back to `~/Projects/Workspace/dotfiles/AGENTS.md`).

## Runtime brain lives in the vault, not in this checkout

Hermes does **not** clone the dotfiles repo. It is provisioned remotely by
`ai/hermes/setup.sh` (curled once) and reads its operating knowledge from the
vault. This file is documentation in the dotfiles repo (per-agent overlay,
ADR-009 parity with `ai/agy/AGY.md`); Hermes's live constitution and SSOT are:

- **Constitution:** `80_agents/hermes-nan/AGENTS.md` (operating law: write-zone,
  sync discipline, bootstrap protocol).
- **Identity + structure map:** `80_agents/hermes-nan/00-context.md`.
- **Shared patterns:** `00_meta/patterns/` — loaded on demand via the
  `pattern-loader` skill.

## Hermes-specific rules (on top of root `AGENTS.md`)

- **Write zone.** Hermes commits **only** within `80_agents/hermes-nan/` (the
  autonomous-agent commit policy, PR #189). It has read access to the whole
  vault — that cross-project brain is the value — but never writes outside its
  workspace.
- **Language.** Chat with Manu in **Spanish from Spain**; **all** vault content
  (files, commit messages) in **English**.
- **Vault access.** Hive MCP via `uvx hive-vault`, pointed at the persistent
  clone (`$HERMES_VAULT_PATH`, default `~/.local/state/hermes/vault`).
- **Secrets.** `GITHUB_TOKEN_KNOWLEDGE` lives in `~/.hermes/.env` (chmod 600) —
  never in chat or vault. Hermes diverges from the dotfiles age-secrets model by
  design (the remote box has neither the dotfiles repo nor the age key).
- **Provisioning.** `ai/hermes/setup.sh` is idempotent and non-interactive; it
  does not touch the local-deploy surface and is never wired into `setup-linux`.

## Model Tier (per root `AGENTS.md` "Model Selection")

- **Default:** `deepseek-v4-flash` (NaN endpoint) — interactive ops.
- **Async / cron:** `qwen3.6` — scheduled, multilingual jobs.

Model identifiers reflect the NaN catalog; verify availability in the Hermes
runtime if a listed model is rejected.
