---
id: "HARNESS-136-model-limit-drift"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-21"
issue: "mlorentedev/dotfiles#1594"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-136-model-limit-drift

## Why

`ai/pi/models.json` hand-declares a context window and an output cap per model
while the providers publish the real ones, and nothing compares the two — so on
2026-09-21 **all seven declared models disagreed with the catalog at once**, in
both directions. The two directions are different failures: a window above the
real limit makes the provider reject a request this file invited, and a cap
below it forfeits paid-for capability in silence (`qwen3.8-flash` shipped 16384
against a real 131072, on the model `qf` routes to). The values survived review
because 2^20 reads as "1M" and no check existed, which is the same shape as
HARNESS-067 one field over: that check asserts a model *id* still resolves and
says nothing about the limits declared beside it.

## What

`dotf doctor` gains a **Model limit drift** section. It reads the declaration
from the checkout and the published limits from opencode's models.dev cache, and
reports every disagreement naming the model, the field, both numbers, and the
consequence. An over-declaration FAILS; an under-declaration WARNS. A model the
catalog does not publish, a limit it does not state, an absent cache, and a run
outside a checkout each produce a SKIP or silence — never a PASS, because
"nothing to compare" and "everything agrees" print the same clean report and
mean opposite things.

## Out of scope

- **Writing.** No `--fix`, and the flag is not even passed in. The values are a
  reviewed repo edit that carries a comment explaining which model changed and
  why; a silent rewrite of a file a human authored is a different decision.
- **A committed catalog snapshot.** It would make CI able to run the check, at
  the cost of adding a second hand-maintained copy of numbers somebody else
  publishes — the exact defect this reports, one indirection further away.
- **Fields other than `contextWindow` and `maxTokens`.** The catalog also
  publishes cost and modalities; only these two change what a request is allowed
  to ask for.
- **The deployed copy.** `checkModelPins` covers deployed routing files. A limit
  finding is only actionable where it can be committed, so this reads the
  checkout.

## Risks / open questions

- **The cache can be stale.** models.dev is refreshed by opencode, not by this
  repo, so a legitimately raised limit reads as an under-declaration until the
  cache updates. Accepted: that direction only WARNs, and the message names both
  numbers so a human settles it in one look.
- **Resolved — provider scoping.** The bare id `qwen3.8-flash` is published by
  `nan` (262144 context) and by a dozen other providers — alibaba, hyper,
  opencode-go, requesty… — at 1000000 or 1048576. (OpenRouter keys its row
  `qwen/qwen3.8-flash`, so it is not among them; the first version of this note
  said it was, corrected in review round 1.) A lookup keyed on the bare model
  id reads the wrong row and reports an honest declaration as four times too
  small; the first implementation pass did exactly that. Lookup is scoped by
  provider and a test fixes it.
- **Resolved — no machine, no check.** CI never sees the cache, so this cannot
  be a CI assertion without the snapshot rejected above. It lives in doctor, the
  same split `checkModelPins` documents.

## Acceptance criteria

- [x] AC1 — the seven declared models match the limits published for their own
      provider, verified against two independent agreeing sources (models.dev
      and <https://nan.builders/docs/models>).
- [x] AC2 — a declaration above the published limit is reported as FAIL and
      names both numbers.
- [x] AC3 — a declaration below the published limit is reported as WARN, and
      never as FAIL.
- [x] AC4 — the same model id under two providers resolves to its own
      provider's row, so a correct declaration under either is not a finding.
- [x] AC5 — an absent catalog cache reports SKIP and never PASS.
- [x] AC6 — a model absent from the catalog, and a limit the catalog does not
      publish, are not findings; when nothing at all was compared the check
      reports SKIP rather than a vacuous PASS.
- [x] AC7 — an unreadable or unparseable declaration FAILS, rather than
      reporting no drift.
- [x] AC8 — the check never writes.

## References

- Bitácora board: `mlorentedev/dotfiles#1594`
- Sibling check: `cli/internal/doctor/checks_model_pins.go` (HARNESS-067, #902) —
  model *id* drift, and the source of the doctor-not-CI and severity-follows-
  consequence reasoning reused here.
- Provider truth: <https://nan.builders/docs/models>; `~/.cache/opencode/models.json`
