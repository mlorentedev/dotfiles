---
id: "HARNESS-093-reviewer-pool-random"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-29"
issue: "mlorentedev/dotfiles#1370"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, review]
template_version: "1.0"
---

# HARNESS-093-reviewer-pool-random

> **Naming**: file lives at `<repo>/specs/HARNESS-093-reviewer-pool-random/proposal.md`. `HARNESS-093-reviewer-pool-random` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

`dotf spec review` always runs the pool's first entry (`nan/deepseek-v4-flash`) unless `--reviewer`
names another, so every adversarial review lands in one NaN rate bucket — the bucket
`model-map.json` already moved PR-Agent away from — and the fallbacks are exercised only by hand
(one of them, the provider-diverse agy arm, had never run once until #1156). Two more
reasoning-class NaN models already live in `ai/pi/models.json` (`glm5.3-flash`, `qwen3.8-flash`:
`reasoning: true`, 1M context) and the gate does not use them. Owner decision 2026-08-29: pure
random selection among the pool by default, both models admitted now (#1370).

## What

- `harness/reviewer-pool.json` gains `nan/glm5.3-flash` and `nan/qwen3.8-flash` (runner `pi`,
  `NAN_API_KEY`). The array is the archive gate's allow-list, so both become signable. Their `why`
  states what is and is not established: reasoning-class by profile, admitted for bucket spread,
  not yet calibrated against a review of known outcome.
- `dotf spec review <id>` with no `--reviewer` draws one pool member uniformly at random
  (`math/rand/v2`, auto-seeded) and says which on the launch line (`random draw` instead of the
  `role`); `--reviewer <pool-id>` still selects a member deliberately (`requested`), which is how a
  run is reproduced on the same model. The chosen id is already recorded in `review.md`
  (`reviewer:`) and `review-request.json`.
- Guard (`tests/reviewer-pool.bats`): unique ids, at least four members, and every `pi` member must
  be a model `ai/pi/models.json` marks `reasoning: true` — a latency-only daily driver cannot enter
  the gate by edit.
- The `adversarial-review` skill's usage line (vault source and committed record) says "a pool
  member drawn at random" instead of "the pool's primary reviewer".

## Out of scope

- Calibrating the two new members against a review of known outcome (e.g. CLI-050 round 1's real
  Major): recorded on their pool entries as not done; a follow-up measurement, not this change.
- Weighted or round-robin draws; a per-spec deterministic draw. Random was the owner's call, and the
  recorded id plus `--reviewer` give reproducibility.
- The local CPU load a review causes (it runs go test, bats and check-doc-paths on the box): a
  property of running reviews concurrently on one machine, independent of the model.

## Risks / open questions

- A random draw can land on `agy/gemini-3.1-pro-high`, whose auth is a login, not a key; a box
  without that login sees the existing "launch died — try another pool member" path. Resolved:
  that is the pool's stated purpose (exercise the fallbacks), and the launcher already names the
  alternative.
- The `role` field (`primary`/`fallback`) no longer steers selection. Kept as documentation of
  provenance; the launch line no longer prints it as if it did. Resolved.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] AC1 — With no `--reviewer`, `ResolveReviewer` returns the member the draw selects (any index
  reachable); with `--reviewer`, the named member, or an error listing the pool.
- [ ] AC2 — The launch line names the chosen id and whether it was drawn or requested; `--dry-run`
  shows the drawn member's command.
- [ ] AC3 — The pool holds the four NaN reasoning models; every `pi` member resolves to a
  `reasoning: true` model in `ai/pi/models.json`; the skill's usage line (vault + record) reads
  "drawn at random".
- [ ] AC4 — On the Windows work box: three `dotf spec review <id> --dry-run` launches show at least
  two different reviewers drawn; a `--reviewer nan/glm5.3-flash` dry-run prints that command.

## References

- Bitácora board: #1370 (HARNESS-093); related HARNESS-071 (#955, the pool), #1156 (agy auth)
- Related: `harness/model-map.json` (NaN bucket note), `ai/pi/models.json` (reasoning flags)
