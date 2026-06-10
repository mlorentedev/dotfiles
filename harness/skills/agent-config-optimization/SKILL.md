---
name: agent-config-optimization
description: "Optimize Hermes Agent configuration for specific use cases — context length tuning, compression settings, tool limits, delegation params. Includes verification and vault backup procedures."
---

# Hermes Config Optimization

When the user wants to tune or optimize Hermes Agent configuration, follow this workflow.

## Prerequisites

1. **Know the model's context limit** — check OpenRouter docs or the provider's model catalog. Don't guess.
2. **Read current config** — `hermes config show` or read `/hermes-home/config.yaml` (or `$HERMES_HOME/config.yaml`).
3. **Check what's already changed** — compare against vault backup.

## Step-by-Step

1. **List all tunable params** — use Python + yaml to dump the full config structure.
2. **Present options grouped by priority** — high (context/compression), medium (guardrails/delegation), low (display/platforms).
3. **Apply one batch at a time** — user prefers one group at a time with pauses.
4. **Verify each change** — check with `hermes config show` + Python verification script.
5. **Backup to vault** — copy config.yaml to vault, commit, push.

## Common Optimizations

### Reasoning Effort

- `agent.reasoning_effort` — controls main conversation model's thinking depth
- `delegation.reasoning_effort` — controls subagent thinking depth
- Valid values: `none` (off), `low`, `medium`, `high`, `xhigh`/`max`
- DeepSeek V4 Flash supports `low`, `medium`, `high`, `max` — set via `hermes config set agent.reasoning_effort high`
- **Remember:** `hermes` binary is at `/opt/hermes/.venv/bin/hermes` — not in system PATH. Use full path or activate venv.
- See `references/reasoning-effort-per-provider.md` for per-provider value mappings.

### Context Window Maximization
- `model.context_length` — set to model's max (e.g., 1000000 for DeepSeek V4 Flash)
- `compression.threshold` — increase to 0.75 (compress at 75% instead of 50%)
- `compression.protect_last_n` — increase to 40
- `compression.protect_first_n` — increase to 5
- `compression.hygiene_hard_message_limit` — increase to 800

### Throughput / Capacity
- `agent.max_turns` — increase from 90 to 120+
- `tool_output.max_bytes` — increase from 50K to 100K+
- `tool_output.max_lines` — increase from 2000 to 4000+
- `delegation.max_concurrent_children` — increase from 3 to 5

### Resilience
- `delegation.max_iterations` — increase from 50 to 80
- `tool_loop_guardrails.hard_stop_after.same_tool_failure` — increase from 8 to 12
- `sessions.retention_days` — increase from 90 to 180
- `checkpoints.enabled` — enable for rollback capability

## Verification Script

```python
import yaml
with open('/hermes-home/config.yaml') as f:
    config = yaml.safe_load(f)
checks = {
    'model.context_length': config.get('model', {}).get('context_length'),
    'agent.max_turns': config.get('agent', {}).get('max_turns'),
    'compression.threshold': config.get('compression', {}).get('threshold'),
    # ... add all changed params
}
for param, value in checks.items():
    print(f"{'✅' if value is not None else '❌'} {param} = {value}")
```

## Vault Backup

**Preferred method: consolidate into `13-config.md`.** Don't create a separate `config-backup.yaml` — it drifts from the actual config and adds maintenance burden. Instead, update `80_agents/hermes-nan/13-config.md` with a "Restore from This File" section containing the exact `hermes config set` commands for each changed parameter.

```bash
# Update 13-config.md with new values + restore commands
cd $HERMES_VAULT_PATH
git add -A
git commit -m "vault: update config with optimization changes"
git push
```

**Fallback: raw YAML backup only if user explicitly requests it.**
```bash
cd $HERMES_VAULT_PATH
git pull --rebase
cp $HERMES_HOME/config.yaml 80_agents/hermes-nan/config-backup.yaml
git add 80_agents/hermes-nan/config-backup.yaml
git commit -m "vault: update config backup with optimization changes"
git push
```

## Important Notes

- **Changes require session restart** — `/restart` (gateway) or exit/re-enter (CLI). Changes do NOT apply mid-conversation.
- **Don't set context_length above model's max** — the model will silently truncate. Verify with provider docs (OpenRouter, NaN catalog).
- **Higher tool_output limits = more tokens consumed** — balance with context_length.
- **checkpoints.enabled = true** creates filesystem snapshots — uses disk space.
- **Always update vault on config change.** `13-config.md` must reflect the new values. `config-backup.yaml` must be refreshed. The vault is the SSOT — if config changes on runtime, update the vault file immediately. Every session ends with a commit to vault.
- **Prefer consolidating in existing files.** Don't create separate config files — put optimization values in `13-config.md` under a section. Only create a separate backup file (`config-backup.yaml`) as a raw YAML copy for faster restore. When the user says "esto deberia ser parte de X no?" — they want consolidation, merge it.
- **User prefers one batch at a time with pauses.** Present options grouped by priority, apply one batch, then pause for questions before the next batch. Ask "¿Te parece bien que aplique estos cambios?" before executing.
- **Verify with both `hermes config show` AND a Python yaml script.** The CLI output is a summary; the full YAML dump catches edge cases. Run the verification script and print all changed params at once.
- **Delegation model optimization.** When the main model is deepseek-v4-flash, set `delegation.model` to `qwen3.6` for subagents — faster and cheaper for simple tasks. Apply via `hermes config set delegation.model qwen3.6`.
- **Always configure a fallback_model.** Set `fallback_model.provider` (e.g., `openrouter`) and `fallback_model.model` (e.g., `deepseek/deepseek-v4-flash`) so the agent has a backup if the primary provider goes down. Apply via `hermes config set fallback_model.provider <provider>`.
- **MCP server env vars need explicit PATH.** When configuring MCP servers in `config.yaml`, add `env: { PATH: /root/.local/bin:/usr/local/bin:/usr/bin:/bin }` and any required config vars like `HIVE_VAULT_PATH: /persist/hermes-vault`. The MCP process does not inherit the parent shell PATH.
- **Report bugs properly, don't work around them.** If a tool has an API issue (wrong params, missing features), create a GitHub issue with the exact test and error. Don't patch around it. The user is the beta tester and wants clean bug reports.
- **SSOT sync rule is non-negotiable.** Every runtime change must be synced to the vault in the same session. The table in `agent-vault-sync` skill lists all update triggers. The auto-sync cron (every 4h) catches missed changes, but the rule is to sync immediately.
