---
id: "AI-033-nan-catalog-additions"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-27"
issue: "mlorentedev/dotfiles#1254"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# AI-033-nan-catalog-additions

> **Naming**: file lives at `<repo>/specs/AI-033-nan-catalog-additions/proposal.md`. `AI-033-nan-catalog-additions` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #1254: AI-033: add qwen3.8-flash and glm5.3-flash to the NaN catalog (pi + opencode) -->

NaN's community added two new reasoning-class models (`qwen3.8-flash`, Alibaba;
`glm5.3-flash`, Zhipu) to its subscription on 2026-08-26. Neither pi's model picker nor
opencode's exposes them: `ai/pi/settings.json`'s curated `enabledModels` and
`ai/opencode/opencode.jsonc`'s `provider.nan.models` map only know the four models that
existed before this date, so a user cannot select either even though the underlying NaN
subscription already serves them.

## What

Both model ids become selectable in pi's TUI model picker and opencode's `/models`
picker, with correct capability metadata (context window, reasoning, modalities) so the
picker and any tooling reading it (opencode's interleaved reasoning renderer) behaves
correctly when either is chosen. Neither becomes a default — this is catalog exposure
only, not a routing or behavior change for existing users.

## Out of scope

- Wiring either model into `defaultModel`/`model`/`small_model` (default stays
  `nan/qwen3.6`, chosen by latency, not by this change)
- Admitting either to `harness/reviewer-pool.json`, `.pr_agent.toml`, or
  `harness/model-map.json`'s tiers — tracked separately in #1244, gated on NaN's
  promotional allocation surviving its community vote (expires end of Aug 2026)
- The `ai/pi/settings.json` deploy-time contract (seed-if-missing) not propagating this
  catalog change to a machine that already has `~/.pi/agent/settings.json` — tracked
  separately in #1247

## Risks / open questions

- NaN's `/v1/models` endpoint does not expose context-window metadata, so the 1M-token
  figures recorded here are the vendors' own published claims (Alibaba, Zhipu), not
  independently measured against NaN's endpoint. Resolved as acceptable: matches how
  every sibling NaN model's `contextWindow` in this repo is already sourced.
- `qwen3.8-flash`'s 1M context is YaRN-extended from a native 262K (Alibaba's own release
  notes); `glm5.3-flash`'s 1M is native (Zhipu's). Documented inline in
  `ai/opencode/opencode.jsonc` so a future reader evaluating either for the reviewer pool
  (#1244) sees the asymmetry rather than assuming parity with `deepseek-v4-flash`/
  `mimo-v2.5`.
- NaN's allocation is explicitly promotional (1B tokens until end of August 2026), ahead
  of a community vote on continued availability. Resolved as acceptable for a picker-only
  change: worst case if the vote drops them, the picker entries go stale and get removed
  in a follow-up, no routing depends on them.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `nan/qwen3.8-flash` and `nan/glm5.3-flash` resolve to a valid entry in
      `ai/pi/models.json` and appear in `ai/pi/settings.json`'s `enabledModels`
- [ ] `ai/pi/README.md`'s model list matches `settings.json`'s `enabledModels`
      (`tests/pi-config.bats`, referential-integrity tests)
- [ ] `ai/opencode/opencode.jsonc` carries both under `provider.nan.models` with
      `reasoning_content` interleaving mapped (`tests/opencode.bats`)
- [ ] Neither model is referenced by `defaultModel` (pi) or `model`/`small_model`
      (opencode)
- [ ] Both models verified live and functional against the real NaN API before merge
      (not just config-valid)

## References

- Bitácora board: #1254 (this spec's work-gate issue)
- Related: #1244 (reviewer-pool/model-map admission, deferred), #1247 (settings.json
  field-level sync, deferred)
- `docs/lessons/lesson-150-a-config-file-the-tool-itself-rewrites-must-be-see.md` —
  why `ai/pi/settings.json` is seed-if-missing (context for the "out of scope" #1247 item)

<!-- archived 2026-08-27 — PR: https://github.com/mlorentedev/dotfiles/pull/1255 -->
