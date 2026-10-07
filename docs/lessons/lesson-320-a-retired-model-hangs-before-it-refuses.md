---
id: "lesson-320-a-retired-model-hangs-before-it-refuses"
type: lesson
status: active
title: "A retired model hangs before it refuses, and only a refusal reaches the fallback"
created: "2026-09-30"
owner: manu
tags: [lesson, nan, pr-agent, ci, models, reviewer]
---

# A retired model hangs before it refuses, and only a refusal reaches the fallback

## What happened

NaN retired `mimo-v2.5`, PR-Agent's primary model, on 2026-09-30. It failed in two phases, measured against `api.nan.builders`:

| Phase | What the model did | What PR-Agent did | What the job reported |
|---|---|---|---|
| Hours before retirement | accepted the request and never answered | waited `ai_timeout` (120 s) on each retry, then published nothing | green `review`, no review on the PR |
| After retirement (from about 05:55Z) | answered `401` "This API key does not have access to the requested model" | fell through to `fallback_models` | a review, from the fallback, with nothing saying the primary was dead |

PR-Agent v0.46.0's `retry_with_fallback_models` falls through on any exception, so a refusal reaches the fallback. A hang consumes the timeout instead, and the run ends with nothing published. Neither phase told anyone that the declared model was gone. `dotf doctor` did report `mimo-v2.5 is bound in model-map.json but NaN does not serve it`, but only on a machine where someone ran it.

Two more facts made it harder to see:

- **`GET /v1/models` is not the catalog the key can call.** It lists what the cluster runs. `minimax-h3` is listed and answers 401, and `glm5.3` needs the premium tier. Only a real call, or NaN's published table, says what is callable.
- **A retirement arrives without notice at the call site.** Seven files bound the model: the workflow, the toml, the reviewer pool, the dispatch chain, pi, opencode and the quota table. Each one had to be found by grep.

## Rule

- Before the review, send one minimal call to each model of the declared chain, with a timeout. Review on the first one that answers. A `::warning::` and a job-summary row name each one that did not, and the job fails when none answers. `scripts/pr-agent-model-preflight.sh` does this. It only skips dead entries in declared order; it never picks a model the chain does not name.
- Treat a hang and a refusal as the same outcome: the model did not answer. Distinguish them in the message (`no answer within Ns` against `HTTP 401`), never in the routing.
- Decide what a key can call by calling it, never from `/v1/models`.
- Re-admit a successor model to the reviewer pool on the same bar its predecessor passed. A rename carries the old evaluation to a model it never measured.

## References

- `scripts/pr-agent-model-preflight.sh`, `tests/pr-agent-model-preflight.bats`
- `.github/workflows/pr-agent.yml`, step `models`
- Spec `specs/AI-045-nan-catalog-alignment/`, issue #1763
- NaN docs: https://nan.builders/docs/models, https://nan.builders/docs/choose-a-model
