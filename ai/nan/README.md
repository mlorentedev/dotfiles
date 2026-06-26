# NaN community — provider config for OpenCode + shell aliases

> **Status:** primary OpenCode provider (SDD-007 consolidation, 2026-05-25).
> **Replaces:** OpenCode Go subscription (cancelled — manual action in Zen dashboard).
> **Coexists with:** OpenRouter (frontier fallback), Ollama at `ollama.kubelab.live` (VPN-only homelab — user-managed; API key slot reserved in `secrets/registry.yaml` as `OLLAMA_API_KEY`, commented until the encrypted file exists).
> **Upstream docs:** https://nan.builders/docs · **Dashboard:** https://cloud.nan.builders/ · **Support:** Discord `#support`.

## Service summary

| Property | Value |
|---|---|
| Base URL | `https://api.nan.builders/v1` |
| Auth header | `Authorization: Bearer sk-...` |
| API style | OpenAI-compatible (chat, embeddings, audio, completions, responses) |
| Rate limit | 100 requests/minute · max 5 concurrent |
| Pricing | Community subscription (fixed, not per-token PAYG) |

## Model catalog

### Chat models (visibles en opencode `/models`)

| Model ID | Context | Multimodal | Use case |
|---|---|---|---|
| `deepseek-v4-flash` | **1M** | text | **Default in opencode** — full-repo reasoning, large diffs, root-cause analysis |
| `qwen3.6` | 256K | text + image | Daily quick-questions via `qq`, multilingual (ES) |
| `gemma4` | 256K | text + image | A/B candidate vs qwen3.6 |

### Non-chat models (NO en opencode picker — usar curl/SDK directos)

opencode's modality schema acepta sólo `text|audio|image|video|pdf` como output. `qwen3-embedding`
con `output: "embedding"` rompía la carga entera del config. Por eso estos 3 NO están listados en
`opencode.jsonc` — accédelos directamente vía la API:

| Model ID | Type | Use case |
|---|---|---|
| `qwen3-embedding` | embeddings (4096-dim) | ES↔EN similitud 0.915 — vault indexing, semantic search |
| `kokoro` | TTS | 67 voice packs (voces `ef_dora`, `af_heart`) |
| `whisper` | STT | 99+ idiomas, ~1× realtime |

Ejemplos curl en https://nan.builders/docs/examples (sección Embeddings, TTS, STT).

Errors to expect: `401` invalid key, `404` unknown model, `429` rate limit, `500` server, `524` timeout (large audio).

## Setup (one-time)

### 1. Get the API key

1. Be a NaN community member (subscription required for access).
2. Go to https://cloud.nan.builders/ → user settings → **API Keys**.
3. Generate a key. Format: starts with `sk-`. **The key is personal and non-transferable** — don't share.

### 2. Encrypt + commit the key

NaN's API key is loaded via the repo's age-based secret system. The encrypted file lives at `sensitive/nan.api-key.secret.age`; the mapping in `secrets/registry.yaml` already exposes it as `NAN_API_KEY`.

```bash
# From the repo root:
age -r "$(age-keygen -y ~/.config/age/key.txt)" \
    -o sensitive/nan.api-key.secret.age
# (then paste your sk-... key, press Ctrl-D)

# Verify the secret can be decrypted + loaded:
. scripts/load-secrets.sh && secrets_refresh
echo "${NAN_API_KEY:0:8}…"   # masked sanity-check
```

### 3. Activate the integration

```bash
# Re-run setup to deploy ~/.config/opencode/opencode.jsonc (which references NaN):
./setup-linux.sh           # Linux
.\setup-windows.ps1        # Windows

# Reload your shell so $NAN_BASE_URL + $NAN_API_KEY are exported:
exec zsh   # or: . ~/.bashrc
```

### 4. Smoke test

```bash
# OpenCode TUI: NaN should appear in /models picker
oc
# /models  → select nan/qwen3.6

# Quick-question alias (shell-agnostic: zsh, bash, pwsh):
qqn "explica brevemente qué es zigzag"

# Raw API:
curl https://api.nan.builders/v1/chat/completions \
  -H "Authorization: Bearer $NAN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"qwen3.6","messages":[{"role":"user","content":"ping"}],"max_tokens":50}'
```

## How it's wired in this repo

| File | Role |
|---|---|
| `secrets/registry.yaml` | exposes `nan.api-key` as `NAN_API_KEY` (registry entry) |
| `sensitive/nan.api-key.secret.age` | encrypted key (user creates, see Setup §2) |
| `ai/opencode/opencode.jsonc` | provider block `nan` (default model + 3 chat models) |
| `.zsh/aliases.zsh` + `.bashrc` + `powershell/profile.ps1` | `qqn` alias + `NAN_BASE_URL` export |

## Workflow — when to use which

Empirically measured with `scripts/nan-bench.sh` (2026-05-25):

| Task | Tool / model | Wall |
|---|---|---|
| Coding interactivo, refactor, daily Q&A | OpenCode TUI / `qq` → `nan/qwen3.6` (**default**) | ~0.5-1.5s |
| Coding A/B vs qwen3.6 | OpenCode `/models` → `nan/gemma4` | ~0.5-1.4s |
| Async/batch jobs (no time pressure) | OpenCode `/models` / `qf` → `nan/deepseek-v4-flash` | ~3-5s (reasoning ON) |
| Long-context >256K | OpenCode `/models` → `nan/deepseek-v4-flash` | ~3s + scales w/ input |
| Frontier (Opus / Sonnet / GPT-4) | OpenCode `/models` → `openrouter/<frontier>` (PAYG) | varies |
| Local / homelab (VPN) | OpenCode `/models` → `ollama/<model>` at `ollama.kubelab.live` (uncomment OLLAMA_API_KEY) | varies |
| Architecture, hard debug, root-cause | Claude Code directly (not OpenCode) | n/a |

## Known limitations & gotchas (doc audit + empirical 2026-05-25)

| Gotcha | Detail | Mitigation |
|---|---|---|
| **deepseek-v4-flash reasoning chain SIEMPRE ON** | +30-180 `reasoning_tokens` por call → 3-10× más lento que qwen3.6 (datos en `scripts/nan-bench.sh`) | Default es qwen3.6; deepseek solo para async/long-ctx |
| **deepseek-v4-flash 100M tokens/mes per-member** | Quota mensual silenciosa | Monitor uso; no rutar TODO el tráfico a deepseek |
| **Rate limits account-wide** | 100 RPM + 5 concurrent total — compartido entre `qq` / TUI / curl / Hermes | `nan-bench.sh` ya throttle-a con `sleep 1`; agentic loops paralelos pegan 429 |
| **Tool-calling solo en qwen3.6** | gemma4 no lista "Tool calling"; qwen3.6 usa XML internal traducido por LiteLLM | Para agentic tool-use, fija a qwen3.6 |
| **`/v1/responses` streaming roto** | Emite solo `response.completed`, no deltas | Usar `/v1/chat/completions` (opencode default) |
| **whisper file cap 25MB / 2min** | Audios largos → HTTP 524. NO WAV | OGG/Opus o MP3 |
| **kokoro 15 RPM** | TTS batch se bloquea | Serializar |
| **Embeddings 60 RPM / batch ≤ 32** | Indexar vault entero te rate-limit-ea | Throttle a 30 RPM, batch=32 |
| **API key personal e intransferible** | No compartir entre máquinas/CI | Cada equipo encripta su propia |
| **No SLA, no deprecation, no billing page** | Community-run, best-effort | OpenRouter como fallback frontier |

## NaN-recommended defaults (per /docs/models)

```json
{
  "temperature": 0.6,
  "top_p": 0.95,
  "max_tokens": "500-16000 (no doc'd cap)",
  "stream": "true on /v1/chat/completions only"
}
```

OpenCode no expone estos como provider-level config; los inyecta por agente. Para curl/SDK directos, fíjalos en el body.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `oc /models` shows no `nan/*` entries | `~/.config/opencode/opencode.jsonc` not deployed | Re-run `./setup-linux.sh` |
| `401 Unauthorized` from `qqn` | `$NAN_API_KEY` not exported in current shell | Reload shell (`exec zsh`), or `. scripts/load-secrets.sh && secrets_refresh` |
| `429 rate limit` errors during agentic loops | NaN's 100 rpm / 5 concurrent cap | Switch to `openrouter/<model>` for the burst, or backoff |
| `524 timeout` on `kokoro` TTS | Large audio request, NaN server timeout | Split input into shorter chunks |
| `NAN_API_KEY` empty after `secrets_refresh` | `sensitive/nan.api-key.secret.age` missing / unreadable | Re-run Setup §2; check `~/.config/age/key.txt` exists |

## References

- Provider docs: https://nan.builders/docs (intro), https://nan.builders/docs/api (API reference), https://nan.builders/docs/examples (per-language examples)
- Spec: `specs/SDD-007-ai-tooling-consolidation/proposal.md`
- Decision record (planned, not yet written): repo `docs/adr/` — NaN as default provider. (The old `adr-013` slot is taken by `adr-013-agent-artifact-deploy-engine.md`.)
