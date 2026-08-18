# AGENTS.md (Hermes Provisioning Overlay)

> Target: **Hermes Agent** (Nous Research on NaN infrastructure, Debian 13).
> Read `AGENTS.md` at repo root FIRST — canonical SSOT. This overlay documents Hermes remote provisioning.
> Live runtime brain and operating law live in the vault at `80_agents/hermes-nan/`.

## Architecture & Boundaries

| Concern | Specification |
|---|---|
| **Write Zone** | Strictly bounded to `80_agents/hermes-nan/` (pre-commit enforced) |
| **Live SSOT** | `80_agents/hermes-nan/` (`context.md`, `AGENTS.md`, `memory.md`, `cronjobs.yaml`) |
| **Vault Access** | Hive MCP (`uvx hive-vault`) against `$HERMES_VAULT_PATH` (`~/.local/state/hermes/vault`) |
| **Secrets** | `GITHUB_TOKEN_KNOWLEDGE` in `~/.hermes/.env` (chmod 600) — never in chat or vault |
| **Language** | Chat: Spanish (Spain). Durable files & commits: English. |
| **Provisioning** | `ai/hermes/setup.sh` (curled once, non-interactive, idempotent) |

## Model Tier (NaN Catalog)

- **Default (Interactive ops):** `deepseek-v4-flash` (1M context)
- **Async / Cron / Low-cost:** `qwen3.6` (unlimited token pool)
- **Fast / Backup:** `mimo-v2.5` / `gemma4`

