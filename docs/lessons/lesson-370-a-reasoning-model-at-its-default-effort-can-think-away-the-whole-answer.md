---
id: "lesson-370-a-reasoning-model-at-its-default-effort-can-think-away-the-whole-answer"
type: lesson
status: active
title: "A reasoning model at its default effort can think away the whole answer"
created: "2026-10-08"
---

# A reasoning model at its default effort can think away the whole answer

## Context
The PR-Agent review pool (AI-045 AC9, #1923) was widened with `glm5.3-flash`, NaN's coding
model, beside mimo, deepseek and Claude Haiku. Its probe answers in a second, so it would have
entered the draw on its first run.

## The Trap
On a review-sized prompt (the #2188 diff, 37K tokens, streamed, 16K output tokens), glm at its
default effort reasoned for 289 s and stopped with `finish_reason: length`: every output token
went to reasoning, NaN closed the stream at its 60,000-reasoning-character ceiling, and the
content was empty. `medium` did the same in 338 s. PR-Agent exits cleanly on an empty answer,
so in CI this is a green attempt that publishes nothing, a quarter of the time. The probe
cannot see it, because a one-line question needs no thinking.

A second default made it worse to fix: PR-Agent sends `reasoning_effort` only to models
LiteLLM marks as reasoning-capable, plus an operator list. LiteLLM does not know glm, so it
received nothing; it does know `deepseek-v4-flash`, so deepseek had been receiving PR-Agent's
own default, `medium`, without anyone choosing it. The effort is one setting for the whole
attempt, not one per model.

## The Solution
Measure a candidate on the real workload before admitting it, not on its probe. At
`reasoning_effort: low`, glm reviewed the 37K-token diff in 285 s and a 97K-token one in
215 s, with about half the ceiling to spare: reasoning did not grow with the prompt. The
workflow sets `CONFIG__REASONING_EFFORT: low` and lists glm in
`CONFIG__ADDITIONAL_REASONING_EFFORT_MODELS`, and a test pins both. Lowering the shared
setting was safe only because the other member that receives it, deepseek, accepts the
parameter and does not act on it (ai/nan/README.md); with a second effort-sensitive member,
the attempt would need splitting instead.
