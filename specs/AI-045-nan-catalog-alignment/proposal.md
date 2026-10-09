---
id: "AI-045-nan-catalog-alignment"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-30"
issue: "mlorentedev/dotfiles#1763"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, nan, pr-agent, reviewer-pool, pi, opencode, model-map]
template_version: "1.0"
---

# AI-045: retire mimo-v2.5, and align every NaN consumer with the catalog the key can call

## Why

<!-- from issue #1763: AI-045: retire every mimo-v2.5 binding and route each consumer by role -->

NaN retired `mimo-v2.5` on 2026-09-30. It answered at 05:44Z and returned `401` ("This API key does not have access to the requested model") from about 05:55Z, and it is gone from `GET /v1/models`. Before that it hung for hours: PR-Agent's primary model accepted the diff and never answered, so every pull request got a green `review` job and no review (#1107). Seven files still bind it: PR-Agent's model, a member of the reviewer pool that signs the archive gate, the `mid` dispatch chain, pi's enabled models, opencode's picker, and the quota table. `dotf doctor` already reports `[FAIL] mimo-v2.5 is bound in model-map.json but NaN does not serve it`, but nothing in CI does. Separately, the repository describes NaN's limits from memory, and several facts disagree with the published docs: the concurrency model, `qwen3.8-flash`'s window, and how each model reads `reasoning_effort`.

## What

**PR 1: retire the dead model, and survive the next retirement.**

- Every binding of `mimo-v2.5` moves to `mimo-v2.6-flash`, NaN's successor. That covers PR-Agent's primary, the reviewer-pool member, the `mid` chain, pi's `enabledModels` and opencode's picker. `mimo-v2.5` leaves `harness/nan-quotas.json`.
- The pool member is re-admitted on the same bar the old one passed, not renamed. The planted-defect test is the `((count++))` exit status under `set -e`, run on 2026-09-30, and `mimo-v2.6-flash` found it with a failing input and the fix.
- The PR-Agent workflow gains a preflight step. It sends one minimal chat call per model in the declared chain (primary, then fallbacks), each with a timeout. The review then runs on the first model that answers, and the job summary and a `::warning::` name every model that did not, with its HTTP status and the file to fix. If no model answers, the job fails with `::error::` before PR-Agent runs. That turns a retired or hanging model into a visible, reviewed PR instead of a green job with no review. The chain stays declared in `.pr_agent.toml`; the preflight only skips dead entries of it, in its order.

**PR 2: align the consumers with NaN's documentation for the `nan_member` plan.**

- The concurrency declaration matches the published model. Each model has its own limit: 7 for deepseek-v4-flash, glm5.3-flash and qwen3.8-flash; 5 for mimo-v2.6-flash, qwen3.6 and gemma4. On top of that, the whole key has a cap of 7 simultaneous requests across all models on the base plan.
- opencode and pi carry NaN's published windows and output caps. `qwen3.8-flash` is served at 1,048,576 tokens, not 262K. Thinking is controlled through `reasoning_effort`, with values each model applies, instead of `chat_template_kwargs.enable_thinking`, which NaN ignores.
- `ai/nan/README.md` states the facts the docs publish and the repository gets wrong or omits:
  - `GET /v1/models` lists what the cluster runs, not what the key can call;
  - `glm5.3` needs the premium tier;
  - deepseek raises any `max_tokens` below 16384 to 16384;
  - a reasoning-only stream is closed at 60,000 characters or 420 seconds, and that closure is not billed.

**Prevention: one guard per failure class this incident exposed (sub-issues of #1763).**

- **AI-045b (#1858).** PR-Agent streams its NaN calls. Measured on 2026-09-30: NaN's edge returns 524 at about 125 s to a non-streamed completion, whatever the model.
- **AI-045c (#1859).** PR-Agent's total wait is bounded by a tested budget. `ai_timeout` times attempts times models, plus setup, stays below the job's `timeout-minutes`. Run 36677077122 waited 14 minutes, and the job timeout ended it.
- **AI-045d (#1860).** A daily canary sends a minimal call to every bound NaN model, with no PR involved. A retirement opens an issue before a PR finds it.
- **HARNESS-067 (#902)** gives provider identity one source. Seven files had to be found by grep to move one model.

**Amendment, 2026-10-01 (#1923): CI review survives a saturated NaN.** The owner chose three remedies on #1923. A shorter timeout is AC6's streaming, because the configured `ai_timeout` never fired (run 36826726168, lesson 327). The other two are new:

- **AC9.** A third attempt on a provider outside NaN, run only when both NaN attempts fail, so a NaN-wide saturation still ends in a review or a reported reason.
- **AC10.** One review at a time across the repository, queued rather than dropped, so parallel PRs stop competing for the bucket each of them needs.

## Out of scope

- The other six repositories that bind `mimo-v2.5` (hive, kubelab, web, pollex, iris, garsync). Each one moves in its own pull request under #1763, or through the shared workflow ADR-042 decides.
- The `glm5.3` premium tier. The key does not have it, so nothing routes to it; it stays documented as unavailable.
- Admitting `qwen3.6` or `gemma4` to the reviewer pool. The pool excludes latency-optimised models by rule.
- `agy` as a PR-Agent backend. PR-Agent reaches NaN through LiteLLM's OpenAI-compatible transport. `agy` is a CLI with login auth and no CI credential, so it serves only the reviewer pool, where it already is.

## Risks / open questions

- `mimo-v2.6-flash` has 5 concurrent requests per model and PR-Agent is its main consumer. Several parallel PRs can exhaust that bucket. The fallback to `deepseek-v4-flash` is the existing answer, and the preflight makes it visible.
- The preflight spends one minimal request per model per run, about 20 to 800 completion tokens depending on the reasoning. That is negligible against the quotas, but it is not zero.
- An early-failover preflight means the review can run on a fallback without anyone choosing it. The job summary names the model that reviewed, and PR-Agent records the same in its own run.
- AC9 reverses a recorded decision: `harness/model-map.json`'s `$comment` lists the openrouter pool as retired. The OpenRouter key authenticates, but its credit is spent (5.03 used of 5.00, measured 2026-10-01). AC9 therefore stays blocked on the owner: fund that provider or name another, then deliver the key with `dotf secrets sync ci`. Until then the step is gated on the credential and never runs.
- **Amended 2026-10-08 (#1923, option 1).** The owner named the provider: Anthropic, paid from Max-plan API credits with no card, no
  auto-reload and a $100 spend limit, and asked that it unload NaN rather than only back it up. So
  `anthropic/claude-haiku-5-5` (on a purpose-named key, `PR_AGENT_ANTHROPIC_API_KEY`) joins the two NaN models in a review
  pool drawn with equal weight among the members that answered, as `dotf spec review` draws. Three deviations from the
  text above. It is a pool, not a third attempt: the drawn provider reviews first and the other is the second attempt,
  with PR-Agent's own NaN chain inside the NaN one. The second attempt is keyed on a measured absence of a published
  review, not on the first attempt's outcome, because `fail_on_tool_errors` cannot tell a model failure from a tool
  failure and a review published late must not be duplicated. And Haiku is admitted without the planted-defect bar NaN
  candidates pass; that bar is owed on its first live run (#1923). The pool is a first-pass gate on every PR, distinct
  from the spec archive gate: `harness/reviewer-pool.json`'s rule against Anthropic reviewers stands unchanged.
  Later the same day, after the live runs, the owner widened it: `glm5.3-flash` joins the NaN chain, so four members
  draw at a quarter each, and Haiku gets parity with NaN's prompt cap (200K, accepting the over-100K price on large
  diffs) and adaptive thinking at `high` effort, within a worst case under $0.25 a review that a test holds. glm
  reviews only at `reasoning_effort: low` (lesson 370).
- **Amended 2026-10-09 (#1923, amendment B), budget $60-65 a month.** The owner wanted Haiku and Sonnet in real use
  rather than as members a uniform draw seldom reaches, inside a stated budget. Three changes. PR-Agent's draw is
  weighted, and the weights live in `harness/reviewer-pool.json` (`pr_agent` blocks: mimo 27, glm 19, deepseek 19,
  Haiku 35), so a share is a declared number rather than one over the member count. Sonnet joins as a risk route: a PR
  at or past 1,500 changed lines, or labelled `deep-review`, reviews first on Sonnet, and falls back to the draw when
  Sonnet does not answer. The threshold was 900 at first; the owner raised it after the first live Sonnet review cost
  $0.37, not $0.13. The allowlist of Anthropic models CI may spend on moves from the workflow into
  `scripts/pr-agent-route.sh`, which fails the draw on any other. And the spec archive gate's rule against Anthropic
  reviewers is replaced by "an Anthropic model never signs alone": the first signature stays another vendor's, Sonnet
  adds a second signature on `risk: high` specs, and Haiku signs as a recorded fallback when the first signers failed
  for a classified reason. The local review reads the same Bitwarden item as PR-Agent under its own name
  (`REVIEW_ANTHROPIC_API_KEY`), so one Console spend limit covers both. The budget arithmetic is in verification.md.
- AC10's `queue: max` is documented for workflow-level groups only, and actionlint 1.7.12 does not know the key. Its PR's own run is the measurement: a workflow GitHub refuses fails visibly before any review. Serialised reviews also mean a busy day waits, up to the job's 31 minutes per PR ahead in the queue.

## Acceptance criteria

- [ ] AC1: no live binding names `mimo-v2.5`. `.pr_agent.toml`, `pr-agent.yml`, `harness/model-map.json`, `harness/reviewer-pool.json`, `harness/nan-quotas.json`, `ai/pi/settings.json` and `ai/opencode/opencode.jsonc` all name `mimo-v2.6-flash` where they named it, and `dotf doctor` reports no NaN quota FAIL.
- [ ] AC2: the PR-Agent workflow's model and fallbacks equal `.pr_agent.toml`'s, and the preflight reads its chain from them. A test fails when the two disagree.
- [ ] AC3: the preflight falls through a model that answers non-2xx or times out, and hands PR-Agent the first model that answers. It fails the job when none answers. It never selects a model outside the declared chain. Tests drive it with a stub transport.
- [ ] AC4: `harness/model-map.json` declares per-model concurrency and the per-key cap as NaN publishes them for the base plan, and the loader validates that shape.
- [ ] AC5: opencode's and pi's NaN windows and output caps equal NaN's published figures. Thinking variants use `reasoning_effort`, and `enable_thinking` appears nowhere.
- [ ] AC6 (#1858): PR-Agent's NaN calls stream, and a test pins the setting. A review of #1856's 42K-token diff completes on the primary, measured with PR-Agent's own prompt.
- [ ] AC7 (#1859): a test fails when the job's `timeout-minutes` does not exceed the sum of the attempts' step `timeout-minutes` plus setup. Amended 2026-10-01: the step bound is the one measured to hold, and `ai_timeout` never fired on a held request (lesson 327).
- [ ] AC8 (#1860): a scheduled workflow probes every model `harness/model-pins.json` lists as bound, and opens or updates one issue when any does not answer. A test drives it with a stub transport.
- [ ] AC9 (#1923): when both NaN attempts fail, PR-Agent reviews once more on a provider outside NaN. A missing credential warns and skips that attempt, never fails the job, and the publish guard reports the last attempt that ran. A test pins the gating and the outcome chain.
- [ ] AC10 (#1923): the review job runs in one repository-wide concurrency group with `cancel-in-progress: false` and `queue: max`, and the per-PR workflow group keeps superseding pushes. A test pins both, and a live run shows GitHub accepts the job-level queue.

## References

- NaN docs, read 2026-09-30:
  - https://nan.builders/docs/models (limits, reasoning control, rate limits)
  - https://nan.builders/docs/choose-a-model (ids, tiers, `/v1/models` semantics)
  - https://nan.builders/docs/pi and https://nan.builders/docs/opencode
- Earlier moves: AI-044 (#1762), AI-046 (#1764), AI-047 (#1766)
- Pin-site registry: `harness/model-pins.json`; reviewer pool rule: `harness/reviewer-pool.json` `$comment`
- ADR-042 (shared CI): the other repositories' PR-Agent binding moves through it
