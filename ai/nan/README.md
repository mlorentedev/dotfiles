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
| Rate limit | 60 requests/minute per key, shared by every model · 1.5M tokens/minute for deepseek-v4-flash, mimo-v2.6-flash, qwen3.6 and gemma4 · simultaneous requests per model: 7 on the base plan (10 on premium) for glm5.3, glm5.3-flash, deepseek-v4-flash and qwen3.8-flash, 5 for mimo-v2.6-flash, qwen3.6 and gemma4; the larger models are undocumented (kimi-k3.1-pro measured at 6 on 2026-11-10) |
| Credits | One pool for every chat model, refilled each billing period: 7,000 credits on the inference plan, 20,000 on premium, then purchased credit packs. Out of credits answers `402`. Balance: the dashboard only. `GET /v1/usage` reports raw tokens per model over 30 rolling days, with no cached split and no credits |
| Pricing | Subscription plus a per-model, per-token-type credit rate (input, cached input, output): https://cloud.nan.builders/tokens. Cached input is 5-50x cheaper than fresh input |

## Model catalog

### Chat models (visibles en opencode `/models`)

Credit rates per 1M tokens (input / cached input / output) and NaN's "typical"
blended rate, read from https://cloud.nan.builders/tokens on 2026-11-10. The
typical rate assumes NaN's cache mix; an uncached single-shot call costs the
input rate.

| Model ID | Context | Credits per 1M (in / cached / out) | Typical | Use case |
|---|---|---|---|---|
| `glm5.3-flash` | 1M | 5.25 / 1.05 / 17.5 | 1.4 | **opencode default and `plan` agent**, NaN's recommendation for coding agents; PR-Agent review pool member since 2026-10-08 (AI-045 AC9), at `reasoning_effort: low` |
| `deepseek-v4-flash` | 1M | 10.5 / 0.21 / 42 | 1.1 | pi default; long-context via `qf`; PR-Agent review pool |
| `mimo-v2.6-flash` | 1M | 7 / 0.14 / 14 | 0.6 | the cheapest flash; PR-Agent review pool member. Replaced `mimo-v2.5`, which NaN retired 2026-09-30 (AI-045, #1763) |
| `qwen3.8-flash` | 1M | 7.5 / 0.8 / 23.5 | 1.9 | picker only: the priciest flash per token (AI-044, #1762 moved it off the default) |
| `qwen3.6` | 262K | no credits | 0 | `qq` quick questions; opencode titles (`small_model`); `model-map` low tier. NaN calls it previous generation; it still answers |
| `gemma4` | 262K | no credits | 0 | A/B candidate vs qwen3.6 |
| `hy4` | 1M | 41.7 / 2.1 / 125 | 5.5 | larger model, picker only; 64K output |
| `deepseek-v4-pro` | 1M | 66 / 2.2 / 198 | 7.7 | larger model, picker only; text only; reasons always (no disable alias) |
| `glm5.3` | 1M | 49 / 1.05 / 154 | 11.5 | larger model, not configured: its own 400M-tokens-per-rolling-4h window answers `429`, and nothing can watch it |
| `kimi-k3.1-pro` | 1M | 100 / 10 / 500 (+100 cache write) | 19.9 | larger model, picker only; NaN's pick for frontier agent reasoning; reasons always (no disable alias) |
| `qwen3.8-max` | 1M | 100 / 12.5 / 300 | 20.2 | larger model, picker only |

The larger models spend credits 5-20x faster than the flash ones: keep them off
per-PR and unattended paths. pi and opencode cap their output below what NaN
serves (131,072 tokens; 65,536 for hy4). Source: https://nan.builders/docs/models
and https://nan.builders/docs/choose-a-model (checked 2026-11-10).

`GET /v1/models` lists what the cluster runs, not what this key can call.
`minimax-h3` is listed but undocumented, and answers this key HTTP 401 ("This
API key does not have access to the requested model"; measured 2026-11-10).

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

### 2. Store the key in Bitwarden

`secrets/registry.yaml` maps `NAN_API_KEY` to the Bitwarden item `nan-api-key` (field `api-key`, folder `Dotfiles/apps`), per ADR-028. Put the `sk-...` key there, then:

```bash
dotf secrets verify                       # resolves every secret, reports OK/MISSING/FAILED
dotf secrets run --only NAN_API_KEY -- <cmd>   # hands the key to one process
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
| `secrets/registry.yaml` | maps Bitwarden item `nan-api-key` to `NAN_API_KEY` (and `HIVE_WORKER_API_KEY`) |
| `ai/opencode/opencode.jsonc` | provider block `nan` (default model + 10 chat models) |
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
| **Thinking is controlled by `reasoning_effort`** | The accepted values are model-specific: glm5.3-flash accepts low/medium/high/max; qwen3.6 and gemma4 also accept none/minimal; deepseek-v4-flash, qwen3.8-flash and mimo-v2.6-flash accept the parameter but do not adjust depth | Use only variants the selected model can apply |
| **DeepSeek reserves reasoning output** | `max_tokens` below 16,384 is raised to 16,384, so a prompt inside that final slice of the context window is rejected | Leave at least 16,384 tokens of output room |
| **Reasoning-only stream ceiling** | A stream that emits no content or tool call is closed after 60,000 reasoning characters or 420 seconds with `finish_reason: length` and a `nan_truncation` marker. The closed reasoning-only turn is not billed. | Lower `reasoning_effort` where supported or restructure the prompt |
| **One credit pool** | Every chat model draws from the period's credits, then from credit packs; spent answers `402`. `GET /v1/usage` reports tokens, not credits, so `dotf doctor` reports usage without judging it | Read the balance on the dashboard; keep the larger models off high-volume paths |
| **Rate limits per key and per model** | 60 RPM per key, shared by `qq` / TUI / curl / agents; concurrency is per model (7 or 5 on the base plan) | Parallel agentic loops hit 429: another model is another lane, up to the 60 RPM key-wide cap |
| **`/v1/responses` streaming roto** | Emite solo `response.completed`, no deltas | Usar `/v1/chat/completions` (opencode default) |
| **whisper file cap 25MB / 2min** | Audios largos → HTTP 524. NO WAV | OGG/Opus o MP3 |
| **kokoro 15 RPM** | TTS batch se bloquea | Serializar |
| **Embeddings 60 RPM / batch ≤ 32** | Indexar vault entero te rate-limit-ea | Throttle a 30 RPM, batch=32 |
| **API key personal e intransferible** | No compartir entre máquinas/CI | Cada equipo encripta su propia |
| **No SLA, no deprecation notice, no billing API** | Community-run, best-effort; the credit balance is on the dashboard only | OpenRouter como fallback frontier |

## NaN-recommended defaults (per /docs/models)

```json
{
  "temperature": 0.6,
  "top_p": 0.95,
  "max_tokens": "model-specific; deepseek-v4-flash raises values below 16384",
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
| `402 Payment Required` | The period's credits and the credit packs are spent | Switch to `qwen3.6` or `gemma4` (no credits) until the period refills, or buy a pack |
| `524 timeout` on `kokoro` TTS | Large audio request, NaN server timeout | Split input into shorter chunks |
| `NAN_API_KEY` empty | Bitwarden item `nan-api-key` missing or the vault is locked | `dotf secrets verify`; see Setup §2 |

## References

- Provider docs: https://nan.builders/docs (intro), https://nan.builders/docs/api (API reference), https://nan.builders/docs/examples (per-language examples)
- Spec: `specs/archive/SDD-007-iac-deploy-strategy/proposal.md` (id `SDD-007-ai-tooling-consolidation`, archived under its folder name)
- Decision record (planned, not yet written): repo `docs/adr/` — NaN as default provider. (The old `adr-013` slot is taken by `adr-013-agent-artifact-deploy-engine.md`.)
