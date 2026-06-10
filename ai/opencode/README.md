# OpenCode — client config

> Primary AI coding agent in this dotfiles. Daily TUI / `oc` alias / `qq` / `qf` / `dbg` shell wrappers.

## What lives here

| File | Role |
|---|---|
| `opencode.jsonc` | SSOT for opencode config. Deploys to `~/.config/opencode/opencode.jsonc` via `setup-linux.sh` / `setup-windows.ps1`. |

OpenCode slash-commands are no longer stored here. They are rendered from the
vault skill records (`harness/skills/`) to `~/.config/opencode/commands/<name>.md`
at deploy time by `scripts/compile-harness.sh --deploy` (SDD-008), honoring each
skill's `targets[]`. The former `commands/` mirror + `skills-to-opencode.sh` were
removed.

## Providers wired here

opencode is a **client**; the providers are documented separately. opencode.jsonc references each:

| Provider | Provider docs | Use case |
|---|---|---|
| **NaN community** (default) | [`ai/nan/README.md`](../nan/README.md) | Daily — `nan/qwen3.6` (256K, multimodal), `nan/deepseek-v4-flash` (1M, async), `nan/gemma4` |
| **OpenRouter** | Provider catalog at [openrouter.ai](https://openrouter.ai/models) | Frontier on-demand (PAYG) — Opus / Sonnet / GPT-4 |
| **Ollama** (homelab) | User-managed, endpoint at `ollama.kubelab.live` (VPN-only) | Local models, slot reserved in opencode.jsonc until `OLLAMA_API_KEY` encrypted |

## Why NaN docs live in `ai/nan/`, not here

NaN is **provider-level** (API key, model catalog, rate limits, troubleshooting) — accessible not only via opencode but also via:

- `qq` / `qf` / `dbg` shell aliases (wrappers over `opencode run` and `curl`)
- `scripts/nan-bench.sh` / `nan-quality-bench.sh` / `nan-debug.sh` (direct curl)
- Any OpenAI-compatible SDK (Python, JS, Go)

Mirroring the existing `ai/claude/`, `ai/copilot/` pattern, each provider/agent owns its own directory. Future Ollama provider docs will live at `ai/ollama/` when wired.

## Quick reference

```bash
oc                                 # TUI — opencode --pure (no MCPs, FAST)
ocfull                             # TUI with MCPs + skills (slower, for agentic tool-use)
qq "..."                           # quick-question via nan/qwen3.6
qf "..."                           # long-context via nan/deepseek-v4-flash
dbg "..."                          # nan/deepseek-v4-flash with reasoning chain visible
opencode run -m nan/gemma4 "..."   # explicit model override
```

**Why `oc` defaults to `--pure`**: empirical bisection 2026-05-25 showed opencode's tool-resolution loop hangs on complex queries when MCPs+skills load 38 tool definitions into the system prompt. `--pure` bypasses the entire loop. Use `ocfull` only when you need agentic tool-use (Hive vault writes, drawio render, context7 docs lookup).

## Known gotchas (cross-OS)

- **`socket` MCP disabled** in opencode.jsonc — `mcp.socket.dev` remote endpoint hangs 30s+ on tool discovery, blocking all chat responses. Re-enable only when upstream confirms latency fix.
- **Non-chat NaN models** (`qwen3-embedding`, `kokoro` TTS, `whisper` STT) are NOT in the `/models` picker — opencode rejects `"embedding"` / `"audio"` as output modalities. Access via curl / OpenAI SDK directly.

## See also

- Spec: [`specs/SDD-007-ai-tooling-consolidation/`](../../specs/SDD-007-iac-deploy-strategy/) — full design rationale + acceptance criteria.
- NaN provider: [`ai/nan/README.md`](../nan/README.md) — model catalog, API key, rate limits, troubleshooting.
- Coexistence with Claude Code: do NOT run `oc` and `claude` in parallel on the same repo until hive MCP adds an auto-commit lockfile (tracked in `hive` repo).
