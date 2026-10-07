---
id: "lesson-331-a-shared-budget-is-watched-at-the-account-not-the-binding"
type: lesson
status: active
title: "A shared budget is watched at the account, not the binding"
created: "2026-10-01"
owner: manu
tags: [lesson, doctor, nan, quota, monitoring]
---

# A shared budget is watched at the account, not the binding

## What happened

AI-047 shipped `dotf doctor`'s NaN quota check over the models `harness/model-map.json` binds. NaN meters per API key, and that key is shared by pi, opencode, the `qq`/`qf` wrappers and CI. On 2026-09-26 `qwen3.8-flash` had reached 83% of its 500M quota. After #1772, nothing in the model map bound it any more, so the check never showed it. pi still routed to it.

The first fix also warned on a metered model absent from `/v1/models`, on the theory that the table was stale. The live run flagged `glm5.3`, which is not retired: `/v1/models` filters premium models by the key's tier.

## Rule

When a budget belongs to the account, watch everything that can spend it: the declared metered set, plus whatever has usage this period, plus the bindings. Do not watch only what one consumer configured. Each consumer's config is a partial view, and enumerating consumers is a list that goes stale. A catalog endpoint scoped to the caller only tells you what the caller can see. Absence from it is evidence about access, not about the catalog.

## Guard

`TestEvaluateWatchesEveryMeteredModelNotOnlyBindings` and `TestCheckNaNQuota_WatchesMeteredModelsNothingBinds` give an unbound model at 83% and expect a WARN. Dropping the union fails both. `TestEvaluateIgnoresAMeteredModelTheKeyCannotSee` pins the tier case.
