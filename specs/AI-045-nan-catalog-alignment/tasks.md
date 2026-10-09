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

- [x] [AC6] AI-045b #1858: force streaming for NaN, pinned in a test. In the workflow env on both attempts, not in
      `.pr_agent.toml`, which PR-Agent reads from the default branch (the introducing PR would not stream)
- [x] [AC6] Measure a real review of a diff of #1856's size (42K tokens) on the primary, streamed. #1856 has merged,
      so the measurement is a `/review` on the largest open PR after this lands; result recorded on #1858. #1938
      (31,381 tokens): streamed, the primary answered in 3 min 19 s (run 36838016719); unstreamed, it failed after
      255 s and the fallback ran 362 s against a 120 s timeout (run 36835026463)
- [x] [AC7] AI-045c #1859: the wait's cause is confirmed (run 36826726168: `ai_timeout: 120` configured, twelve
      silent minutes, no timeout logged; lesson 327). The invariant test landed with #1913 as "the job outlives both
      attempts": the job's timeout exceeds the attempts' step bounds plus setup
- [x] [AC8] AI-045d #1860: `dotf harness canary` (`cli/internal/nanprobe`) probes the map's NaN models and every
      `harness/model-pins.json` site, each on its own API (rerank answers 404 on chat), one at a time, retrying an
      unavailable one once. Stub-transport tests in Go; `scripts/model-canary.sh` keeps one issue, tested under `bash -e`
- [x] [AC8] Real sibling, local: against NaN on 2026-10-01, nine models, 26 s; it found `mimo-v2.5` (401) still
      offered by the deployed `~/.pi/agent/settings.json` `enabledModels`
- [x] [AC8] Real sibling, CI: a `workflow_dispatch` run of `model-canary.yml` after merge; run id recorded on #1860 (run 36887489906, green, 6/6 answered)
- [x] [AC9] #1923: a review pool across providers. `scripts/pr-agent-route.sh` draws, with equal weight, among the
      members that answered their probe (mimo-v2.6-flash, deepseek-v4-flash, glm5.3-flash on NaN; `anthropic/claude-haiku-5-5`, gated
      on its credential). The other provider is the second attempt, run only on a measured absence of a published review
      (`scripts/pr-agent-publish-guard.sh --probe`); `unknown` never runs it. `vars.PR_AGENT_PROVIDER` overrides the draw
      (`nan-only` stops all Anthropic spending). Tests pin the draw, the override, the gating, the outcome chain, the
      model, the input/output caps and the single credential per step
- [x] [AC9] Key synced to CI from the branch registry, after a live probe of the key answered 2xx (2026-10-08)
- [x] [AC9] Live run, Anthropic route: `PR_AGENT_PROVIDER=anthropic`, Haiku reviewed first and published, NaN skipped
      (run 37879933613 attempt 1)
- [x] [AC9] Live run, NaN route: override removed, the draw picked `openai/mimo-v2.6-flash`, which published, and the
      Anthropic second attempt was skipped (run 37879933613 attempt 2). A draw, not `PR_AGENT_PROVIDER=nan`: the same
      route, reached the way production reaches it
- [x] [AC9] Owner decisions, 2026-10-08: glm5.3-flash joins the NaN chain (four members at equal weight), and Haiku gets
      the NaN attempt's 200K prompt cap, 32K output and adaptive thinking at `high` effort. LiteLLM 1.103.0 (PR-Agent
      v0.47.0's pin) measured forwarding `thinking` and `output_config` unchanged; the budget test bounds the worst review
      under $0.25
- [x] [AC9] glm5.3-flash measured on review-sized prompts before admission: no review at its default effort or `medium`,
      a review at `low` on 37K and 97K tokens (lesson 370); the NaN attempt sends `low`, pinned by a test
- [x] [AC9] Live run, Haiku with thinking: drawn with no override, the step log carries PR-Agent's `Using adaptive
      thinking for model anthropic/claude-haiku-5-5 with output_config effort 'high'` and a review is published
      (run 37885509177)
- [x] [AC9] Live run, glm5.3-flash: drawn with no override, `reasoning_effort` low in the log, a complete review published
      in 74 s (run 37886959259)
- [x] [AC9] Amendment B (2026-10-09): the weighted draw from the pool's `pr_agent` blocks, the Sonnet risk route (1,500
      changed lines or the `deep-review` label, read with `gh api` because `issue_comment` carries neither), the
      allowlist in `scripts/pr-agent-route.sh`, the routed model and effort in both Anthropic steps, run details on every
      attempt, `retry_same_model_on_timeout = false` on NaN. Tests pin the exact share of every point of the shipped
      pool's weight, the risk route and its fallback, the allowlist, a price row and cost ceiling per allowed model, and
      the pool's NaN members equal to the preflight's chain
- [x] [AC9] Amendment B, archive gate: `signs` and `vendor` in the pool, `review-second.md` required on `risk: high`,
      `--second` and `--fallback-reason` on `dotf spec review`, and pool loading refusing an Anthropic first signer
      (commit 77c89fd5)
- [x] [AC9] Live run, Sonnet risk route: #2188 itself (3,682 changed lines) at `ready_for_review` reviewed first on
      Sonnet with adaptive thinking, published, and the review's run details name the model (run 37907390670)
- [ ] [AC9] Live run, second attempt: the first real failure of a first attempt after merge that runs the other
      provider; run id on #1923 (a failure cannot be forced from CI without spending the shared NaN pool)
- [x] [AC10] #1923: failing test for the repository-wide job queue, then the job-level `concurrency` block
- [x] [AC10] Live run of the PR that adds the queue shows GitHub accepts it: run 36835280780 started its `review` job and published a review (an unknown key fails the workflow before any job starts)
- [x] [AC10] On `main`, two reviews that overlap in time both complete and neither is cancelled; record the run ids on #1923. A cancelled review run there means the job-level queue is ignored: revert the block (runs 36881054024, 36881389377, 36881441749 queued together, ran in turn, none cancelled; #1923 comment 5934615584)

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
