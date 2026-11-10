---
id: "lesson-570-a-provider-billing-change-retires-the-check-not-the-numbers"
type: lesson
status: active
title: "A provider's billing change retires a quota check; updating its numbers would not"
created: "2026-11-10"
---

# A provider's billing change retires a quota check; updating its numbers would not

## Context

`dotf doctor`'s NaN section (AI-047, #1766) compared `GET /v1/usage` tokens per
model with a declared table of monthly per-model quotas
(`harness/nan-quotas.json`) and WARNed at 80%. In November 2026 NaN replaced
those quotas with one pool of credits per billing period, priced per model and
per token type (input, cached input, output), and added four larger models that
cost 5-20x the flash rate.

## Problem

The obvious edit was to refresh the table: new numbers, new date. That would
have kept a check that is wrong in every term:

- **The denominator is gone.** There is no per-model limit left to divide by.
- **The numerator is the wrong unit.** `/v1/usage` reports raw tokens over 30
  rolling days, with no cached split. Cached input now costs 5-50x less than
  fresh input, so one model's last-period tokens ranged from about 1,500
  credits to about 16,000, depending on a cache share the API does not report.
- **No endpoint holds the truth.** `/v1/credits`, `/v1/balance`, `/v1/limits`
  and `/v1/account` all answered 404. The balance exists only on the dashboard.

A refreshed table would have produced confident percentages of nothing.

## Solution

The thresholds and the table were deleted. The section keeps the half that is
still true from the API alone: a binding NaN does not serve FAILs, and each
model's tokens are reported as INFO with the window they cover, followed by a
line saying tokens are not credits. The spend guard moved to where it can hold:
the larger models stay off per-PR and unattended paths, and pi and opencode cap
their output below what NaN serves.

Two measurements made while doing this are recorded in `harness/model-map.json`.
Both were done on the base plan.

- **One model:** 12 parallel requests got 7 through.
- **Three models:** 13 parallel requests got 12 through in one run and 7 in the
  other.

So another model is another lane, but a key-wide limit that NaN does not
document can still answer 429.

## Takeaway

When a provider changes **what** it meters, re-derive the check from the API's
current answers before touching its constants. If the API cannot produce the
verdict, delete the verdict and keep only what it can still prove.
