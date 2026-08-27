---
tags: [spec, tasks, templates]
created: "2026-08-27"
---

# Tasks - AI-033-nan-catalog-additions

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

> **Retroactive note**: the config additions were implemented and functionally verified
> against the live NaN API before this spec folder existed — the diff crossed the
> spec-gate's 50-LOC threshold organically (140 LOC across 5 files) and the gate correctly
> refused the push. Tasks below record what was actually done, in the order it happened,
> not a plan written before implementation.

## Setup

- [x] Branch created from `origin/main` in an isolated worktree (`feat/nan-catalog-additions`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC5] Live-probed both model ids against `api.nan.builders` via `scripts/nan-debug.sh`
      before touching any config — confirmed both real and reasoning-class
      (`reasoning_content` present) on a smoke prompt
- [x] [AC1] Added `qwen3.8-flash` and `glm5.3-flash` entries to `ai/pi/models.json`
      (`providers.nan.models`)
- [x] [AC1] Added `nan/qwen3.8-flash` and `nan/glm5.3-flash` to
      `ai/pi/settings.json`'s `enabledModels`
- [x] [AC2] Updated `ai/pi/README.md`'s model list to match, plus a documentation note on
      the promotional-allocation caveat and the YaRN-vs-native context nuance
- [x] [AC3] Added both models to `ai/opencode/opencode.jsonc`'s `provider.nan.models`,
      mirroring the `mimo-v2.5` entry shape (`interleaved`, `limit`, `modalities`,
      `variants`)
- [x] [AC3] Updated `tests/opencode.bats`: model-count assertions (4 -> 6) and the
      `interleaved` DX-004 AC1 count assertion (4 -> 6)
- [x] [AC5] Re-probed both models against this repo's own planted
      `((count++))`/`set -e` bug (the same defect that admitted `mimo-v2.5` to
      `harness/reviewer-pool.json`) — both diagnosed it correctly
- [x] [AC4] Confirmed by inspection: neither model referenced by `defaultModel`
      (`ai/pi/settings.json`) or `model`/`small_model` (`ai/opencode/opencode.jsonc`)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test or
      recorded manual verification
- [x] Every acceptance criterion has a matching entry in `features.json` with a
      non-vacuous verification command
- [x] Type checks pass (`go build ./... && go vet ./...` in `cli/`, unaffected by this
      diff but re-run to confirm no incidental breakage)
- [x] Lint passes (pre-commit hooks: secrets scan, bats @test names, doc-path check,
      message format)
- [x] No unrelated changes in the diff — isolated worktree branched fresh off
      `origin/main`, patch-applied to include only the 5 catalog/test files
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/AI-033-nan-catalog-additions/features.json`):

```json
[
  {
    "id": "AI-033-nan-catalog-additions-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
