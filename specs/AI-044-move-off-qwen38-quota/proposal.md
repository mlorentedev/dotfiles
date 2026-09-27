---
id: "AI-044-move-off-qwen38-quota"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-26"
issue: "mlorentedev/dotfiles#1762"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, nan, opencode, model-map]
template_version: "1.0"
---

# AI-044: move routed traffic off qwen3.8-flash, and fix two NaN bindings

## Why

opencode routed its default, title and plan traffic to `qwen3.8-flash`, which NaN meters at 500M tokens a month. By 2026-09-26 it had spent 415M (83%, read from `GET /v1/usage`), and a spent quota answers `402` until the month resets, so opencode would stop answering before October. Two bindings were also wrong against NaN's docs: `model-map.json` named the rerank service `qwen3-rerank` (401; the id NaN serves is `rerank`), and opencode declared `qwen3.8-flash` at 1M context when NaN serves 262K, which NaN warns makes the model reject requests.

## What

- opencode's `model` and `agent.plan` route to `glm5.3-flash` (2B a month, NaN's recommendation for coding agents); `small_model`, which titles every session, routes to `qwen3.6`, which has no monthly quota.
- `model-map.json`: the low tier and chain start at `qwen3.6`, then `glm5.3-flash`, then `claude:haiku`; `services.rerank` names `rerank`.
- opencode's context windows equal pi's for every NaN model both carry, held by a test.
- `deepseek-v4-flash` declares image input.
- `ai/nan/README.md` and `.claude/CLAUDE.md` state the routing, quotas and limits NaN documents today.

## Out of scope

- The `mimo-v2.5` to `mimo-v2.6-flash` migration and the move from `enable_thinking` to `reasoning_effort` (AI-045, #1763).
- A quota alarm reading `/v1/usage`, and a live check that every NaN id the map names is served (AI-047, #1766).
- `qq`/`qf` stay on `qwen3.6`/`deepseek-v4-flash`; pi's default stays `deepseek-v4-flash`.

## Risks / open questions

- `glm5.3-flash` latency as a default: measured 0.79-0.91s on three trivial requests (one cold 5.3s outlier), level with `qwen3.6` and `gemma4`.
- The owner chose the default model by accepting the recommendation; it is named in the PR body so it can be vetoed.

## Acceptance criteria

- [x] AC1: opencode's `model` is `nan/glm5.3-flash` and `small_model` is `nan/qwen3.6`.
- [x] AC2: opencode and pi declare the same context window for every NaN model both carry.
- [x] AC3: `model-map.json` routes no tier or chain head to `qwen3.8-flash`, and names the rerank service `rerank`.

## References

- NaN models and quotas: https://nan.builders/docs/models, https://nan.builders/docs/opencode
- Pin-site registry: `harness/model-pins.json`
