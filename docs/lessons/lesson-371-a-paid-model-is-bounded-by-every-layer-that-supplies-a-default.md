---
id: "lesson-371-a-paid-model-is-bounded-by-every-layer-that-supplies-a-default"
type: lesson
status: active
title: "A paid model is bounded by every layer that supplies a default"
created: "2026-10-08"
---

# A paid model is bounded by every layer that supplies a default

## Context
AI-045 AC9 (#1923) adds Claude Haiku 5.5 to the PR-Agent review pool, beside the NaN models,
paid from plan credits under a spend limit. The Console offers no per-key model allowlist, so
the workflow step is the only place where the model, the prompt size and the answer size are
decided.

## The Trap
Setting `CONFIG__MODEL` looks like the whole configuration. Three other layers each supply a
value of their own, and none of them fails loudly:

- **LiteLLM's output cap.** For a Claude model it has no entry for, LiteLLM 1.103.0 (the version
  PR-Agent v0.47.0 pins) sends `max_tokens` from `DEFAULT_ANTHROPIC_CHAT_MAX_TOKENS`, default
  4096. Adaptive thinking counts against that cap, so a review can be cut short with a clean exit.
- **PR-Agent's temperature.** PR-Agent sends `temperature=0.2` unless the model is listed in
  `config.no_temperature_models`. Haiku 5.5 rejects a non-default temperature with a 400.
- **The secret's name.** A key exported as `ANTHROPIC_API_KEY` is picked up by any Claude Code
  process that inherits the environment, which bills the key for work it was never meant to pay
  for. The name decides who can spend it, so it names the purpose:
  `PR_AGENT_ANTHROPIC_API_KEY`, mapped to `ANTHROPIC__KEY` in one step only.

## The Solution
Read the pinned versions' source for every default that reaches the request, and set each one
explicitly: `DEFAULT_ANTHROPIC_CHAT_MAX_TOKENS` (16000 when this was written, 32000 since the
owner gave Haiku adaptive thinking), the model listed in `CONFIG__NO_TEMPERATURE_MODELS`,
`CONFIG__MAX_MODEL_TOKENS` (first sized under the 100K-token price step, then raised to the NaN
attempt's 200000 by the owner's choice of parity, which accepts the higher price on large diffs),
and an empty `CONFIG__FALLBACK_MODELS` so a failure never escalates to a pricier model. Whatever
the values, the rule is the same: each is a stated number, and a test bounds the worst review it
allows. `tests/pr-agent-config.bats` pins each value, and
`harness/pr-agent-upstream-contract.json` pins PR-Agent's LiteLLM handler (`litellm_ai_handler.py`), so an Action bump
that changes these paths fails the contract check instead of the bill.

A spend limit is the backstop. The budget is whatever the defaults allow, so set them yourself.
