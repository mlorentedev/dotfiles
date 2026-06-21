---
name: agent-lifecycle
description: "Complete operational lifecycle of a Hermes Agent instance — bootstrap/recovery, vault integration, config optimization, cronjob scheduling, and container deployment. Absorbs agent-bootstrap, agent-config-optimization, cronjob-management, and vault-integration (the standalone agent-config-optimization skill is retired into this skill's Section 3)."
version: 1.0.0
platforms: [linux]
metadata:
  hermes:
    tags: [bootstrap, recovery, config, cron, vault, container, ops, lifecycle]
    related_skills: [vault-sync, mcp-integration, hermes-agent]
---

# Agent Lifecycle — Setup, Config, Scheduling & Recovery

Complete operational lifecycle of a Hermes Agent instance. This is the umbrella skill for everything that keeps an agent running, recoverable, and properly configured.

## Sections

1. [Bootstrap & Recovery](#section-1-bootstrap--recovery)
2. [Vault Integration](#section-2-vault-integration)
3. [Config Optimization](#section-3-config-optimization)
4. [Cronjob Management](#section-4-cronjob-management)
5. [Container Deployment](#section-5-container-deployment)

---

## Section 1: Bootstrap & Recovery

When an agent runs on infrastructure that could be lost (cloud VM, container, Kubernetes pod), the vault must be the single source of truth for agent state.

### Durable State Files

Every agent should maintain these files under `80_agents/<agent-name>/`:

| File | Purpose | Sync Trigger |
|------|---------|-------------|
| `memory.md` | Snapshot of agent's internal persistent memory | On every memory(add|replace|remove) |
| `skills.md` | Registry of custom skills created by this agent | On every skill_manage create/patch/delete |
| `cronjobs.md` | Active Hermes cron jobs | On every cronjob create/pause/resume/remove |
| `config.md` | Model, provider, toolsets, gateways | On every config change |

**Design principle:** Store what you need to reconstruct, not a snapshot of everything. `config.md` should document the ESSENTIAL provider/model config (4-10 lines), not a full dump of `config.yaml`.

### Bootstrap Protocol (6 Phases)

1. **Prerequisites** — Run `setup.sh`, ensure cron is installed, set up git hooks and system crons
2. **Restore memory** — Read `memory.md`, import each entry via memory(add)
3. **Restore skills** — Read `skills.md`, re-create each custom skill via skill_manage(create)
4. **Restore cron jobs** — Read `cronjobs.md`, re-create each job via cronjob(action='create')
5. **Restore config** — Read `config.md`, apply model/provider/toolsets/gateway config
6. **Context recovery** — Browse `sessions/`, check bitácora GitHub Project for active issues, resume operations

### Validation Script

```bash
bash /hermes-home/scripts/validate.sh
```

Checks: git status, hooks, system cron, durable state files, local AGENTS.md, scripts directory, Hive MCP, cron job count.

### What NOT to do during bootstrap
- Don't ask for approval on auto-session writing — if there was activity, write it. If not, stay silent.
- Don't interleave documentation with implementation. Finish the full implementation FIRST, THEN document.
- Don't store full `config.yaml` in the vault — it diverges over time. Store essential snippets only.

### See also
- `references/bootstrap-checklist.md` — Printable checklist for fire drills
- `references/validation-examples.md` — Example outputs from validate.sh

---

## Section 2: Vault Integration

Connect Hermes Agent to an Obsidian vault via Hive MCP for persistent cross-agent knowledge sharing.

### When to use
- Setting up a new Hermes Agent node in a multi-agent ecosystem with a shared knowledge vault
- Integrating an agent with Obsidian vault for persistent cross-agent memory
- Creating agent-specific workspace folders in a vault with scope-based project resolution

### Prerequisites
- Access to the vault repository (GitHub fine-grained token, read-write Contents)
- Hive MCP installed (`uvx hive-vault`) with vault scopes configured
- Tailscale/SSH access to the agent host

### Step 1: Configure Hive MCP scope

```python
vault_scopes = {
    "projects": "10_projects",
    "meta": "00_meta",
    "work": "50_work",
    "agents": "80_agents",    # ← new scope for agent inboxes
}
```

**Pitfall:** The `agents` scope must be LAST in the dict — auto-scan resolves first-match over insertion order.

### Step 2: Clone the vault

```bash
git clone https://x-access-token:${GITHUB_TOKEN_KNOWLEDGE}@github.com/<owner>/<repo>.git /persist/hermes-vault
```

**Rule:** All temporary files go to `/tmp/`. Never leave artifacts in `/opt/hermes/` or home directory.

### Step 3: Create the agent constitution

Create `~/.hermes/.agents/<agent-name>/AGENTS.md` with explicit rules for:
- Language discipline: Chat in user's language, vault content in English
- Git discipline: Always `git pull --rebase` before reading/writing, `git push` after
- Hive MCP first: Always prefer `vault_write`, `vault_search`, `vault_query` over native file tools
- Temp files: Everything goes to `/tmp/`
- Security: Never share tokens/keys over chat

### Key Pitfalls
- **MCP server installed but not registered:** Installing `hive-vault` does NOT register it with Hermes. Must also add to `config.yaml` under `mcp_servers:` or run `hermes mcp add hive --command uvx --args hive-vault`.
- **Hive "Project not found":** The directory doesn't exist yet. Create it with `mkdir -p` before calling `vault_write(operation=create)`.
- **Scope not in vault_scopes:** If `agents:hermes-nan` doesn't resolve, the scope hasn't been added to Hive's config.
- **Token scoping:** A fine-grained token scoped to one repo cannot write to another repo. Use separate tokens.

### See also
- `references/hive-mcp-api.md` — Full API reference for vault_write, vault_list, vault_health, vault_commit, vault_search, vault_query
- `references/session-lifecycle.md` — When/how to write session files
- `references/cronjob-registry.md` — How to document Hermes cronjobs in the vault

---

## Section 3: Config Optimization

When the user wants to tune or optimize Hermes Agent configuration.

### Prerequisites
1. **Know the model's context limit** — check provider docs. Don't guess.
2. **Read current config** — `hermes config show` or read `/hermes-home/config.yaml`.
3. **Check what's already changed** — compare against vault backup.

### Step-by-Step
1. **List all tunable params** — use Python + yaml to dump the full config structure.
2. **Present options grouped by priority** — high (context/compression), medium (guardrails/delegation), low (display/platforms).
3. **Apply one batch at a time** — user prefers one group at a time with pauses.
4. **Verify each change** — check with `hermes config show` + Python verification script.
5. **Backup to vault** — update `13-config.md` with restore commands for each changed parameter.

### Common Optimizations

#### Reasoning Effort
- `agent.reasoning_effort` — controls main conversation model's thinking depth
- `delegation.reasoning_effort` — controls subagent thinking depth
- Valid values: `none`, `low`, `medium`, `high`, `xhigh`/`max`
- **Delegation model optimization:** When the main model is deepseek-v4-flash, set `delegation.model` to `qwen3.6` for subagents — faster and cheaper.

#### Context Window Maximization
- `model.context_length` — set to model's max (e.g., 1000000 for DeepSeek V4 Flash)
- `compression.threshold` — increase to 0.75 (compress at 75% instead of 50%)
- `compression.protect_last_n` — increase to 40
- `compression.protect_first_n` — increase to 5

#### Throughput / Capacity
- `agent.max_turns` — increase from 90 to 120+
- `tool_output.max_bytes` — increase from 50K to 100K+
- `delegation.max_concurrent_children` — increase from 3 to 5

#### Resilience
- `delegation.max_iterations` — increase from 50 to 80
- `sessions.retention_days` — increase from 90 to 180
- `checkpoints.enabled` — enable for rollback capability

Additional tunables worth knowing: `compression.hygiene_hard_message_limit` (raise to ~800), `tool_output.max_lines` (2000 → 4000+), and the resilience guardrail `tool_loop_guardrails.hard_stop_after.same_tool_failure` (8 → 12).

### Important Notes
- **Changes require session restart** — `/restart` (gateway) or exit/re-enter (CLI). Changes do NOT apply mid-conversation.
- **The `hermes` binary is not on PATH** — it lives at `/opt/hermes/.venv/bin/hermes`. Use the full path or activate the venv.
- **Don't set context_length above model's max** — the model will silently truncate.
- **Higher tool_output limits = more tokens consumed** — balance with context_length.
- **Always configure a fallback_model.** Set `fallback_model.provider` and `fallback_model.model` so the agent has a backup if the primary provider goes down.
- **MCP server env vars need explicit PATH.** When configuring MCP servers, add `env: { PATH: /root/.local/bin:/usr/local/bin:/usr/bin:/bin }` and any required config vars (e.g. `HIVE_VAULT_PATH`).
- **SSOT sync rule is non-negotiable.** Every runtime change must be synced to the vault in the same session — the `agent-vault-sync` skill lists all update triggers; the 4h auto-sync cron only catches misses, it is not the rule.
- **Prefer consolidating in existing files.** Put optimization values in `13-config.md` under a section. **Do NOT create a separate `config-backup.yaml`** — it drifts from the live config. Use `13-config.md` with a "Restore from This File" section holding the exact `hermes config set` commands for each changed parameter; keep a raw YAML copy only if the user explicitly asks for a fast-restore artifact.
- **Report bugs, don't work around them.** If a tool has an API issue (wrong params, missing feature), file a GitHub issue with the exact test + error instead of patching around it.
- **Verify with BOTH `hermes config show` AND a Python `yaml` script** that prints every changed param at once — the CLI output is a summary; the full dump catches edge cases.

### See also
- `references/reasoning-effort-per-provider.md` — Per-provider reasoning effort value mappings
- `agent-vault-sync` skill — the full table of runtime→vault update triggers

---

## Section 4: Cronjob Management

Create, update, and manage scheduled cron jobs via the `cronjob` tool.

### Two Execution Modes

#### 1. Agent-driven (default: `no_agent: false`)
The LLM agent runs the prompt each tick. Skills can be attached. Use for reasoning-heavy tasks.

```
cronjob action=create \
  prompt="Analyze the latest logs and summarize issues" \
  schedule="0 9 * * *" \
  skills=["skill1", "skill2"]
```

#### 2. Script-only (`no_agent: true`)
Runs a script on schedule, delivers stdout verbatim. Zero LLM cost. Use for health checks, monitoring, data collection.

```
cronjob action=create \
  script="vault-validate.sh" \
  schedule="*/5 * * * *"
  no_agent=true
```

### ⚠️ CRITICAL: Script Format Pitfall

With `no_agent: true`, the `script` field MUST be a **filename only** (e.g., `vault-validate.sh`), NOT inline script content with a shebang.

**WRONG** — this fails because the scheduler interprets the shebang as part of the filename:
```
script: "#!/usr/bin/env bash\nset -euo pipefail\ncd /path\nbash script.sh"
```

**CORRECT** — place the script in `/persist/hermes-home/scripts/` and reference by name:
```
script: "vault-validate.sh"
```

The scheduler looks for the file at `/persist/hermes-home/scripts/<script_name>`.

### Script Location

All scripts for `no_agent: true` jobs go in `/persist/hermes-home/scripts/`.

Scripts should:
1. Use `set -euo pipefail`
2. Exit 0 on success (silent delivery with `no_agent: true`)
3. Exit non-zero and print to stdout on failure (triggers error alert)
4. Be self-contained (no external state assumptions beyond what's documented)

### Model Selection Rules

| Task type | Model | Why |
|-----------|-------|-----|
| Agent-driven complex tasks | `deepseek-v4-flash` | Strong reasoning, 1M context |
| Creating cron jobs | `qwen3.6` | No token limit, always available |
| Simple extraction/classification | `gemma4` | Ultra-lite, high volume |
| Async cron runs | `qwen3.6` | Default for cron, no quota |

### Common Patterns

#### Health check (silent on success)
```bash
#!/usr/bin/env bash
set -euo pipefail
# Check something...
if [ condition ]; then
  echo "✅ All good"
  exit 0  # silent delivery
fi
echo "❌ Problem detected"
exit 1  # error alert
```

### Troubleshooting

| Problem | Cause | Fix |
|---------|-------|-----|
| "Script not found: #!/usr/bin/env bash" | Script field contains inline content | Move script to `/persist/hermes-home/scripts/`, set `script` to filename only |
| Job never runs | Invalid schedule format | Check cron syntax, use `cronjob action=list` to verify |
| Script produces no output | With `no_agent: true`, exit 0 = silent | Add explicit echo statements to report status |
| Wrong environment | Cron jobs run in fresh session | Scripts must be self-contained, use `workdir` if needed |

### See also
- `references/cronjob-script-format-error.md` — Detailed incident transcript of the inline-script pitfall

---

## Section 5: Container Deployment

When Hermes runs in a container without systemd (Kubernetes pods, NaN platform), there are critical architecture differences.

### Detecting container environment

```bash
test -e /run/systemd/system && echo "systemd" || echo "no systemd"
```

### Starting services without systemd

Instead of `systemctl enable --now <service>`, start the daemon directly:

```bash
/usr/sbin/sshd
ss -tlnp | grep 22
sshd -T | grep -E 'permitrootlogin|passwordauthentication|pubkeyauthentication'
```

### Critical rules
- **`systemctl enable --now` will fail** with "systemd is not running" — do not attempt.
- **policy-rc.d blocks starts** — avoid invoke-rc.d, service, or systemctl calls.
- **Services do NOT survive container restarts** — the container's init system handles that.
- **Cron daemon also needs direct start** if the bootstrap uses it: `/usr/sbin/cron -f &`

### Vault SSOT discipline for scripts

The vault is the single source of truth for all operational scripts. Every script in the system must have a corresponding copy in `80_agents/<agent-name>/scripts/`.

**Direction rule:** vault → runtime. Scripts are copied FROM vault TO runtime. If a script exists only in runtime, copy it to the vault first, then propagate. Never modify a script in runtime without updating the vault copy.

**Bootstrap reference pattern:** System-level scripts that must live at a fixed path outside the vault (e.g., `/persist/hermes-startup.sh` called by the container entrypoint) are COPIED to the vault as documentation/bootstrap reference. They are NOT symlinked — symlinks break if the vault isn't cloned yet. The vault copy serves as the recovery source: "on a new server, copy this file to the fixed path."

**Pitfall — vault docs can drift from reality.** The `12-cronjobs.md` file may document system crons that don't actually exist. Always verify runtime state against vault documentation:
```bash
crontab -l 2>/dev/null || echo "no crontab"
pgrep -x vault-pull-daemon >/dev/null && echo "daemon running" || echo "daemon NOT running"
```
If the vault says something exists but it doesn't, fix the runtime OR update the vault — never leave them out of sync.

**Pitfall — containers without crontab.** On Debian trixie containers, the `cron` package may be installed but its binaries (`crontab`, `cron`) may be missing from the filesystem. `apt-get install --reinstall cron` may not restore them. Instead of fighting crontab, use a background daemon loop with PID file for idempotency:

```bash
#!/usr/bin/env bash
# vault-pull-daemon.sh — Simple loop daemon for vault git pull
set -euo pipefail
VAULT_PATH="${HERMES_VAULT_PATH:-/persist/hermes-vault}"
PIDFILE="/tmp/vault-pull-daemon.pid"

if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
  exit 0
fi
echo $$ > "$PIDFILE"

while true; do
  cd "$VAULT_PATH"
  git pull --rebase --quiet 2>/dev/null || { git rebase --abort 2>/dev/null || true; }
  sleep 300
done
```

Start it from `hermes-startup.sh` with idempotency check. The daemon is documented in `12-cronjobs.md` as a system-level entry.

### Container binary disappearance (Debian trixie)

On Debian trixie containers, installed packages may have their **binaries removed from the filesystem** between container rebuilds or `apt-get` operations. The package metadata (`dpkg -l`) shows the package as installed, but `which <binary>` returns nothing. This affects `tailscale`, `tailscaled`, `sshd`, `crontab`, and potentially others.

**The wipe also strips package FILES (libs/data) while dpkg still marks them installed.** The worst case is the `systemd` stack — openssh's postinst calls `systemd-sysusers`, which fails (missing binary OR missing `libsystemd-shared-*.so`) and leaves openssh half-configured, which wedges **all** apt. Same shape with `hicolor-icon-theme` (google-chrome's postinst calls `xdg-icon-resource`, needs `/usr/share/icons`). `apt-get install --reinstall` can fail here with `Internal Error, No file name for X`; the fix is `apt-get download` the .deb(s) + `dpkg -i` directly (dependency libs BEFORE the dependent package, so unpack restores `.so` files before configure). See `ensure_systemd_stack` / `ensure_icon_theme` / `restore_wiped` in `hermes-startup.sh`. (HERMES-026, 2026-06-09)

**Fix:** Add `ensure_bin()` / `ensure_lib()` to the startup script. The canonical implementation
lives in `scripts/hermes-startup.sh` (the SSOT — don't copy-paste it, it drifts). Pattern:

```bash
ensure_bin() {   # reinstall a package if its binary vanished
  pkg="$1"; bin="$2"
  command -v "$bin" >/dev/null 2>&1 || apt_reinstall "$pkg"
}
ensure_lib() {   # reinstall if a shared lib vanished — rebuild the cache FIRST (stale -p lies)
  pkg="$1"; lib="$2"; ldconfig 2>/dev/null
  ldconfig -p | grep -q "$lib" || { apt_reinstall "$pkg"; ldconfig; }
}
# apt_reinstall prefers a VERIFIED install; it falls back to --allow-unauthenticated ONLY if the
# apt keyring itself was wiped, and logs a warning. Never disable signature checks unconditionally.
```

Use them before starting any service:
```bash
ensure_bin tailscale tailscale
ensure_bin tailscale tailscaled
ensure_bin openssh-server sshd
ensure_lib libwrap0 libwrap.so.0
ensure_lib libwtmpdb0 libwtmpdb.so.0
```

**Also needed for sshd:** Create `/run/sshd` before starting:
```bash
mkdir -p /run/sshd && sshd
```

**Pitfall:** `command -v` may find the binary at a different path than expected (e.g., `/usr/bin/tailscale` vs `/usr/sbin/tailscaled`). Always use `command -v` rather than hardcoded paths in the startup script.

**Pitfall — SSH connection closed immediately (`kex_exchange_identification: Connection closed`):** On Debian trixie containers, OpenSSH 10.x uses a separate `sshd-session` binary (at `/usr/lib/openssh/sshd-session`) for per-connection handling. This binary links against `libwrap.so.0` (TCP Wrappers) and `libwtmpdb.so.0`. If these libraries are missing from the container's overlay filesystem, `sshd-session` fails to load and sshd closes the connection without sending a banner. The symptom is:
```
kex_exchange_identification: Connection closed by remote host
```
**Diagnosis:** Check with `ldd /usr/lib/openssh/sshd-session | grep "not found"`. **Fix:** Install `libwrap0` and `libwtmpdb0`, then `ldconfig`. Add `ensure_lib` calls for both before starting sshd.

### Tailscale in container mode

Kubernetes pods usually lack a `/dev/net/tun` device. Tailscale must run in **userspace networking mode**:

```bash
tailscaled --tun=userspace-networking \
  --state=/var/lib/tailscale/tailscaled.state \
  --socket=/var/run/tailscale/tailscaled.sock &
tailscale up --login-server=https://your.headscale.com --accept-routes=false
```

### Container persistence patterns

| Storage Type | Characteristics | What goes there |
|-------------|----------------|-----------------|
| Volatile (rootfs, tmpfs, emptyDir) | Ephemeral — recreated on pod restart | Temp files, cache, cloned repos |
| Persistent (PVC, hostPath, CSI) | Survives pod restarts | Vault clone, Tailscale state, SSH keys, Hermes data |

### Startup persistence chain

The full chain that ensures services survive pod restarts:

```
Container entrypoint (/opt/hermes/docker/entrypoint.sh)
  └─ Executes /persist/hermes-startup.sh (if exists and root)
       ├─ ensure_bin() — reinstalls missing binaries (tailscale, sshd)
       ├─ tailscaled + tailscale up — connects to Headscale
       ├─ sshd — starts SSH server
       └─ vault-pull-daemon — git pull --rebase every 5 min
```

**Key points:**
- The entrypoint modification is on persistent storage (`/opt/` is on `/dev/vdb`), so it survives pod restarts
- `/persist/hermes-startup.sh` is the single entry point for all startup services
- The startup script is COPIED to the vault (`scripts/hermes-startup.sh`) as bootstrap reference — NOT symlinked
- On a new server: copy from vault to `/persist/hermes-startup.sh`, make executable, add entrypoint hook
- Documented in `AGENTS.md` under "Startup & Persistence" section

### Workflow discipline: implement then document

When doing multi-step tasks (install, configure, bootstrap):
1. **Finish the full implementation FIRST** — all commands run, all files modified, all verifications pass
2. **THEN document** — vault session note, server docs, other records

Do NOT interleave: no "let's document this partial state" mid-implementation.

### Ongoing ops: install a service → persist to vault

Every time you install, configure, or enable a system service:
1. **Install** the package
2. **Verify** the service is running
3. **Check runtime config**
4. **Record in vault** — update `80_agents/hermes-nan/20-servers.md`
5. **Git commit + push**

**Pitfall — patch tool + markdown tables:** The `patch` tool may add `||` (double pipe) prefixes instead of preserving single `|` pipes in markdown tables. **Workaround:** After patching a table, always `read_file` the full file to verify formatting. If the table broke, rewrite the entire file with `write_file` instead.

### See also
- `references/container-quirks.md` — NaN platform and container-specific notes
- `references/entrypoint-patching.md` — Patching the Hermes Docker entrypoint on NaN
- `references/ssh-connection-closed-libwrap.md` — SSH `kex_exchange_identification` failure from missing libwrap/libwtmpdb
