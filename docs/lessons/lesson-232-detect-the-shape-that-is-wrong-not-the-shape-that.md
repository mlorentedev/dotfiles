# Lesson 232 — detect the shape that is wrong, not the shape that is merely absent

**Date:** 2026-08-26
**Context:** HARNESS-067 / #902 — the model-pin drift guard.
**Category:** guards, false positives, registries

## What happened

The guard checks that model ids pinned around the repository resolve in
`harness/model-map.json`. Its registry's own `$comment` says, at length, that a
**catalog is not routing** — a picker may offer models nothing routes, and
reporting those would fire on a recorded decision (#1244).

The first implementation then reported every catalog id the map does not route.
Its first run against the real machine:

```
[WARN] "nan/gemma4" resolves to nothing the routing registry declares
[WARN] "nan/deepseek-v4-flash-0731" resolves to nothing the routing registry declares
[WARN] "openrouter/deepseek/deepseek-v4-pro" ...
```

The first line is wrong. `gemma4` is a live NaN model that nothing routes —
exactly as legitimate as `qwen3.8-flash` and `glm5.3-flash`, whose deliberate
absence from the map is tracked in #1244. The warning the check was written to
avoid was written into the check, by the session that wrote the warning.

## The lesson

**"Not declared" is the cheap predicate and it is almost always the wrong one.**
Absence from a registry is the normal state of most things. A guard keyed on it
fires on every legitimate extension, and a guard that cries wolf gets muted,
which is worse than no guard.

Find the predicate that means something is *wrong* rather than merely *absent*.
Here it was mechanical once the question was asked properly:

| Id | Absent from the map? | Wrong? |
|---|---|---|
| `nan/gemma4` | yes | **no** — unrouted, which is what a catalog is for |
| `nan/deepseek-v4-flash-0731` | yes | **yes** — a declared id (`deepseek-v4-flash`) plus a date stamp |
| `openrouter/minimax/minimax-m3` | yes | **yes** — names a provider this repo retired |

`-0731` is a *frozen snapshot* of a model still alive under its rolling name. It
goes stale by construction, and the tool rejects it outright (`No models match
pattern`). `gemma4` bears no such relation to anything. Two narrow, defensible
rules replaced one broad, indefensible one — and the check went from 5 findings
with 1 false positive to 4 findings with none.

## Two things that made the difference

**Running it against real data, early.** The false positive was invisible in
review and obvious in the first live run. A guard's first execution on real
state *is* its design review; budget for changing the design afterwards.

**A regression test that names the legitimate case.**
`TestModelPinsDoesNotReportAnUnroutedCatalogModel` asserts `gemma4`,
`qwen3.8-flash` and `glm5.3-flash` are all silent. Guards get broadened under
pressure — someone will want to catch "one more thing" — and the test is what
makes the narrowing survive that.

## The uncomfortable part

This is the same family as `lesson-230` (a config that parses is not a config
the consumer reads) and `lesson-231` (declaration is not effect). The variant
worth naming: **writing the warning does not exempt you from the mistake.** The
prose describing the trap and the code falling into it were authored minutes
apart. Prose is not a guard; only the test is.
