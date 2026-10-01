---
id: lesson-327
type: lesson
status: active
created: "2026-10-01"
owner: manu
tags: [lesson, pr-agent, nan, litellm, timeouts, ci]
---

# 327 — A configured timeout is not a bound until a run shows it firing

## What happened

#1923 asked for a shorter LiteLLM request timeout, so that a NaN model holding a review-sized request would fail in minutes and leave the retry its full bound. Before writing a smaller number, the failed run was read for the current one.

Run 36826726168 (#1887, a 48,736-token diff) logs `"ai_timeout": 120` in PR-Agent's effective config. The next line, "Tokens: 48736 ... returning full diff", is at 06:49:22. After it, the step logged nothing for twelve minutes, until its own `timeout-minutes` stopped it. The retry on the second model did the same. No timeout was logged, and no same-model retry, although `retry_same_model_on_timeout` is true. The 120 s bound was configured and never fired.

So a smaller `ai_timeout` would have changed nothing. Why it does not fire is not measured. PR-Agent passes it to LiteLLM as `timeout`, and the request was not streamed. The lever that changes the transport is streaming (AI-045 AC6, #1858). It also removes the measured edge cut at about 125 s on non-streamed answers.

## Rule

- Before lowering a timeout, find a run where the current one fired. If none exists, the number is a declaration, and a smaller declaration fixes nothing.
- The step's `timeout-minutes` is the bound that is measured to hold here. Size it from the slowest successful run, and keep the job's timeout above the sum of the attempts (the "job outlives both attempts" test).
- When a remedy changes the transport (streaming), measure it on one real review before closing the ticket. The config test proves the setting is present, not that it works.

## Guard

`tests/pr-agent-config.bats`: "every attempt streams its NaN calls" pins the setting on both attempts and checks its substring against the base URL each step sends. Whether streaming bounds the wait is AC6's live measurement, recorded on #1858.

That the pinned build reads the three keys was checked against the code, not the logs. The action never logs `litellm_ai_handler` records (zero in run 36834986053), so the absence of "Using streaming mode" there proves nothing. PR-Agent at the pinned commit was installed in a scratch venv and run with the workflow's env. Its `_acompletion` was replaced with a function that records the keyword arguments. With the three `LITELLM__*` keys set, the call went out with `stream=True`, provider `openai`, and `api_base` NaN's. Without them, it went out with `stream=None`. Run the same check again whenever the pin moves, because the condition is upstream code.
