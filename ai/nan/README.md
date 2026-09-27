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
| Rate limit | 60 requests/minute per key · 1.5M tokens/minute per model · 5-7 concurrent per model |
| Quota | Monthly tokens per model (see the catalog); a spent quota answers `402` until the month resets. Own usage: `GET /v1/usage` |
| Pricing | Community subscription (fixed, not per-token PAYG) |

## Model catalog

### Chat models (visibles en opencode `/models`)

| Model ID | Context | Quota / month | Use case |
|---|---|---|---|
| `glm5.3-flash` | 1M | 2B | **opencode default and `plan` agent** — NaN's recommendation for coding agents |
| `deepseek-v4-flash` | 1M | 3B | pi default; long-context via `qf` |
| `qwen3.6` | 262K | none (per-minute limits only) | `qq` quick questions; opencode titles (`small_model`); `model-map` low tier |
| `gemma4` | 262K | none | A/B candidate vs qwen3.6 |
| `qwen3.8-flash` | 262K | 500M | picker only since 2026-09-26: it reached 83% of its quota as the default (AI-044, #1762) |
| `mimo-v2.5` | 1M | 1B | pr-agent reviewer; `mimo-v2.6-flash` replaces it (AI-045, #1763) |

Every chat model reads images. Source: https://nan.builders/docs/models (checked 2026-09-26).

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
dotf secrets verify
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
| `ai/opencode/opencode.jsonc` | provider block `nan` (default model + 6 chat models) |
| `.zsh/aliases.zsh` + `.bashrc` + `powershell/profile.ps1` | `qqn` alias + `NAN_BASE_URL` export |

## Workflow — when to use which

Empirically measured with `scripts/nan-bench.sh` (2026-05-25):

| Task | Tool / model | Wall |
|---|---|---|
| Coding, refactor, daily Q&A | OpenCode TUI → `nan/glm5.3-flash` (**default**); `qq` → `nan/qwen3.6` | ~0.8s (measured 2026-09-26) |
| Coding A/B vs qwen3.6 | OpenCode `/models` → `nan/gemma4` | ~0.5-1.4s |
| Async/batch jobs (no time pressure) | OpenCode `/models` / `qf` → `nan/deepseek-v4-flash` | ~3-5s (reasoning ON) |
| Long-context >256K | OpenCode `/models` → `nan/deepseek-v4-flash` | ~3s + scales w/ input |
| Frontier (Opus / Sonnet / GPT-4) | OpenCode `/models` → `openrouter/<frontier>` (PAYG) | varies |
| Local / homelab (VPN) | OpenCode `/models` → `ollama/<model>` at `ollama.kubelab.live` (uncomment OLLAMA_API_KEY) | varies |
| Architecture, hard debug, root-cause | Claude Code directly (not OpenCode) | n/a |

## Known limitations & gotchas (doc audit + empirical 2026-05-25)

| Gotcha | Detail | Mitigation |
|---|---|---|
| **Thinking is controlled by `reasoning_effort`** | `chat_template_kwargs.enable_thinking` is ignored (same reasoning output, measured 2026-09-26) | Send `reasoning_effort` (`none` to disable); the opencode variants move to it in AI-045 (#1763) |
| **Monthly quota per model** | 500M to 3B depending on the model; spent answers `402` until the month resets | Read `GET /v1/usage`; do not route a high-volume path to a 500M model (AI-047, #1766) |
| **Rate limits per key and per model** | 60 RPM per key, shared by `qq` / TUI / curl / Hermes; 5-7 concurrent per model | Parallel agentic loops hit 429: back off, or spread across models |
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
| `401 Unauthorized` from `qqn` | `$NAN_API_KEY` not exported in current shell | Reload shell (`exec zsh`), or check with `dotf secrets verify` |
| `429 rate limit` errors during agentic loops | NaN's 60 rpm per key / per-model concurrency cap | Switch to `openrouter/<model>` for the burst, or backoff |
| `402 Payment Required` | That model's monthly quota is spent | Switch model until the month resets; check `GET /v1/usage` |
| `524 timeout` on `kokoro` TTS | Large audio request, NaN server timeout | Split input into shorter chunks |
| `NAN_API_KEY` empty after `secrets_refresh` | `sensitive/nan.api-key.secret.age` missing / unreadable | Re-run Setup §2; check `~/.config/age/key.txt` exists |

## References

- Provider docs: https://nan.builders/docs (intro), https://nan.builders/docs/api (API reference), https://nan.builders/docs/examples (per-language examples)
- Spec: `specs/archive/SDD-007-iac-deploy-strategy/proposal.md` (id `SDD-007-ai-tooling-consolidation`, archived under its folder name)
- Decision record (planned, not yet written): repo `docs/adr/` — NaN as default provider. (The old `adr-013` slot is taken by `adr-013-agent-artifact-deploy-engine.md`.)
