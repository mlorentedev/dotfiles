---
id: "AI-045-nan-catalog-alignment"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-30"
issue: "mlorentedev/dotfiles#1763"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, nan, pr-agent, reviewer-pool, pi, opencode, model-map]
template_version: "1.0"
---

# AI-045: retire mimo-v2.5, and align every NaN consumer with the catalog the key can call

## Why

<!-- from issue #1763: AI-045: retire every mimo-v2.5 binding and route each consumer by role -->

NaN retired `mimo-v2.5` on 2026-09-30. It answered at 05:44Z and returned `401` ("This API key does not have access to the requested model") from about 05:55Z, and it is gone from `GET /v1/models`. Before that it hung for hours: PR-Agent's primary model accepted the diff and never answered, so every pull request got a green `review` job and no review (#1107). Seven files still bind it: PR-Agent's model, a member of the reviewer pool that signs the archive gate, the `mid` dispatch chain, pi's enabled models, opencode's picker, and the quota table. `dotf doctor` already reports `[FAIL] mimo-v2.5 is bound in model-map.json but NaN does not serve it`, but nothing in CI does. Separately, the repository describes NaN's limits from memory, and several facts disagree with the published docs: the concurrency model, `qwen3.8-flash`'s window, and how each model reads `reasoning_effort`.

## What

**PR 1: retire the dead model, and survive the next retirement.**

- Every binding of `mimo-v2.5` moves to `mimo-v2.6-flash`, NaN's successor. That covers PR-Agent's primary, the reviewer-pool member, the `mid` chain, pi's `enabledModels` and opencode's picker. `mimo-v2.5` leaves `harness/nan-quotas.json`.
- The pool member is re-admitted on the same bar the old one passed, not renamed. The planted-defect test is the `((count++))` exit status under `set -e`, run on 2026-09-30, and `mimo-v2.6-flash` found it with a failing input and the fix.
- The PR-Agent workflow gains a preflight step. It sends one minimal chat call per model in the declared chain (primary, then fallbacks), each with a timeout. The review then runs on the first model that answers, and the job summary and a `::warning::` name every model that did not, with its HTTP status and the file to fix. If no model answers, the job fails with `::error::` before PR-Agent runs. That turns a retired or hanging model into a visible, reviewed PR instead of a green job with no review. The chain stays declared in `.pr_agent.toml`; the preflight only skips dead entries of it, in its order.

**PR 2: align the consumers with NaN's documentation for the `nan_member` plan.**

- The concurrency declaration matches the published model. Each model has its own limit: 7 for deepseek-v4-flash, glm5.3-flash and qwen3.8-flash; 5 for mimo-v2.6-flash, qwen3.6 and gemma4. On top of that, the whole key has a cap of 7 simultaneous requests across all models on the base plan.
- opencode and pi carry NaN's published windows and output caps. `qwen3.8-flash` is served at 1,048,576 tokens, not 262K. Thinking is controlled through `reasoning_effort`, with values each model applies, instead of `chat_template_kwargs.enable_thinking`, which NaN ignores.
- `ai/nan/README.md` states the facts the docs publish and the repository gets wrong or omits:
  - `GET /v1/models` lists what the cluster runs, not what the key can call;
  - `glm5.3` needs the premium tier;
  - deepseek raises any `max_tokens` below 16384 to 16384;
  - a reasoning-only stream is closed at 60,000 characters or 420 seconds, and that closure is not billed.

## Out of scope

- The other six repositories that bind `mimo-v2.5` (hive, kubelab, web, pollex, iris, garsync). Each one moves in its own pull request under #1763, or through the shared workflow ADR-042 decides.
- The `glm5.3` premium tier. The key does not have it, so nothing routes to it; it stays documented as unavailable.
- Admitting `qwen3.6` or `gemma4` to the reviewer pool. The pool excludes latency-optimised models by rule.
- `agy` as a PR-Agent backend. PR-Agent reaches NaN through LiteLLM's OpenAI-compatible transport. `agy` is a CLI with login auth and no CI credential, so it serves only the reviewer pool, where it already is.

## Risks / open questions

- `mimo-v2.6-flash` has 5 concurrent requests per model and PR-Agent is its main consumer. Several parallel PRs can exhaust that bucket. The fallback to `deepseek-v4-flash` is the existing answer, and the preflight makes it visible.
- The preflight spends one minimal request per model per run, about 20 to 800 completion tokens depending on the reasoning. That is negligible against the quotas, but it is not zero.
- An early-failover preflight means the review can run on a fallback without anyone choosing it. The job summary names the model that reviewed, and PR-Agent records the same in its own run.

## Acceptance criteria

- [ ] AC1: no live binding names `mimo-v2.5`. `.pr_agent.toml`, `pr-agent.yml`, `harness/model-map.json`, `harness/reviewer-pool.json`, `harness/nan-quotas.json`, `ai/pi/settings.json` and `ai/opencode/opencode.jsonc` all name `mimo-v2.6-flash` where they named it, and `dotf doctor` reports no NaN quota FAIL.
- [ ] AC2: the PR-Agent workflow's model and fallbacks equal `.pr_agent.toml`'s, and the preflight reads its chain from them. A test fails when the two disagree.
- [ ] AC3: the preflight falls through a model that answers non-2xx or times out, and hands PR-Agent the first model that answers. It fails the job when none answers. It never selects a model outside the declared chain. Tests drive it with a stub transport.
- [ ] AC4: `harness/model-map.json` declares per-model concurrency and the per-key cap as NaN publishes them for the base plan, and the loader validates that shape.
- [ ] AC5: opencode's and pi's NaN windows and output caps equal NaN's published figures. Thinking variants use `reasoning_effort`, and `enable_thinking` appears nowhere.

## References

- NaN docs, read 2026-09-30:
  - https://nan.builders/docs/models (limits, reasoning control, rate limits)
  - https://nan.builders/docs/choose-a-model (ids, tiers, `/v1/models` semantics)
  - https://nan.builders/docs/pi and https://nan.builders/docs/opencode
- Earlier moves: AI-044 (#1762), AI-046 (#1764), AI-047 (#1766)
- Pin-site registry: `harness/model-pins.json`; reviewer pool rule: `harness/reviewer-pool.json` `$comment`
- ADR-042 (shared CI): the other repositories' PR-Agent binding moves through it
