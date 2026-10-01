---
tags: [spec, tasks, templates]
created: "2026-09-29"
---

# Tasks - AI-045-nan-catalog-alignment

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `fix/retire-mimo-v25` (PR 1)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (the three listed are accepted risks, not blockers)

## Implementation

### PR 1: retire mimo-v2.5, and survive the next retirement

- [x] [AC3] Write `tests/pr-agent-model-preflight.bats` with a stub `curl`: primary answers, primary 401, primary hangs, none answers, never outside the chain, bare id probed, key on stdin only, job summary, usage errors. Watched it fail (exit 127, script absent).
- [x] [AC3] Implement `scripts/pr-agent-model-preflight.sh` until 9/9 pass; shellcheck clean. Mutation: moving the key to argv fails the stdin test.
- [x] [AC2] [AC3] Wire the `models` step into `.github/workflows/pr-agent.yml`: one sparse checkout for both scripts, `CONFIG__MODEL` and `CONFIG__FALLBACK_MODELS` from its outputs, the no-review guard skipped when it failed.
- [x] [AC2] Test that `DECLARED_MODEL` / `DECLARED_FALLBACK_MODELS` equal the toml's `model` / `fallback_models`. Mutation: a mismatched model fails it.
- [x] [AC1] Move every live binding to `mimo-v2.6-flash`: `.pr_agent.toml`, `harness/model-map.json` (`chains.mid`), `harness/reviewer-pool.json` (re-admitted on the planted-defect bar), `ai/pi/settings.json`, `ai/opencode/opencode.jsonc`; drop `mimo-v2.5` from `harness/nan-quotas.json`. Tests and docs follow.
- [x] [AC3] Real-dependency sibling `tests/pr-agent-model-preflight-real.bats` (BUG-055 pairing), and the remedy split by status class (401/403/404 fix the chain; 402/429 quota or concurrency; 5xx/000 NaN did not serve it)
- [x] [AC1] `tests/pi-nan-package.bats` against a real pi: the pinned package registers `mimo-v2.6-flash` as reasoning-class with the same 1M window opencode declares.

### PR 2: align the consumers with NaN's docs

- [x] [AC4] Failing loader test for per-model concurrency plus the per-key cap in `harness/model-map.json`
- [x] [AC4] Declare them as NaN publishes them for `nan_member`, and validate the shape in the loader
- [x] [AC4] Update the real semaphore regression to saturate 5 dispatchable
  slots (`7` key-wide minus `2` interactive reserve), not the retired `5-2`
  budget
- [x] [AC5] Failing test: no `enable_thinking` in opencode, and NaN windows and output caps equal the published figures
- [x] [AC5] Move thinking variants to `reasoning_effort`; `qwen3.8-flash` to 1,048,576; README facts

### Prevention (one PR each)

- [ ] [AC6] AI-045b #1858: force streaming for NaN in `.pr_agent.toml`, pin it in a test, measure a real review of #1856's diff
- [ ] [AC7] AI-045c #1859: confirm the 14-minute wait's cause, then add the time-budget invariant test
- [ ] [AC8] AI-045d #1860: daily canary workflow over `harness/model-pins.json`, with a stub-transport test and a real sibling

## Closing

- [x] Every acceptance criterion of PR 1 (AC1-AC3) is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Lint passes (shellcheck, actionlint)
- [x] No unrelated changes in the diff
- [ ] `verification.md` filled in (PR 1 and PR 2 done; prevention PRs pending)
- [x] PRs opened referencing this spec folder: PR 1 #1856, PR 2 #1916

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/AI-045-nan-catalog-alignment/features.json`):

```json
[
  {
    "id": "AI-045-nan-catalog-alignment-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
