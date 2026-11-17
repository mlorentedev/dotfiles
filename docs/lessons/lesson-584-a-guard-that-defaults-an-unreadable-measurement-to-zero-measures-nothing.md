---
id: "lesson-584-a-guard-that-defaults-an-unreadable-measurement-to-zero-measures-nothing"
type: lesson
status: active
title: "A guard that defaults an unreadable measurement to zero is asserting the absence of a limit"
created: "2026-11-11"
---

# A guard that defaults an unreadable measurement to zero is asserting the absence of a limit

## Context

AI-055 (#3301), found in review of AI-054 (#3289). `dotf harness check` enforces
that PR-Agent's prompt clip leaves room for the answer inside the context
window: `max_model_tokens + max_output_tokens < custom_model_max_tokens`. The
reserve it adds is the answer cap it reads from the reviewer workflow.

## Problem

It read that cap with `largestOutputCap`, which walks the steps and returns `0`
when a step declares no `CONFIG__MAX_OUTPUT_TOKENS` or declares one the int
parser cannot read. Zero is not a neutral answer here — it is the strongest
possible claim, that the answer occupies no room, so `clip + 0 < window` passes
on a clip that fills the entire window. The check reported OK on the one input it
could not evaluate, and the input was a single typo away: a quoted value, an
expression, or a missing line.

The same function already failed closed four lines away on an expression
`ai_timeout`, so within one guard two measurements disagreed about which
direction uncertainty points. That asymmetry is the tell: when data is missing,
the code did not choose a default, it inherited the zero value of a Go `int`,
and the zero value happened to sit on the passing side.

## Solution

`answerReserve` returns the reserve *and* a list of problems, and it
distinguishes "declared but unreadable" (report the step and the raw string) from
"no step declares it" (report the variable). When the reserve is unknown, no clip
line prints at all: an OK line about a comparison that never happened is worse
than no line, because it is indistinguishable from a comparison that held.

The general form: a measurement a guard cannot read is a **gap**, never a zero,
and never the default value of its type. Decide the direction explicitly, and if
two checks in one guard decide differently, one of them is wrong.

## Takeaway

When a guard reads a number it will compare against, the absent case is a branch
to write on purpose — and the safe default is the one that fails, not the one
that compiles to zero.