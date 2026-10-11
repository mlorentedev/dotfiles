---
id: "dotfiles-pr-agent-reviewer"
type: runbook
status: active
tags: [dotfiles, ci, review, pr-agent, nan, github-actions, cost]
created: "2026-08-18"
owner: manu
---

# PR-Agent reviewer on NaN inference — dotfiles

> The second autonomous reviewer on `mlorentedev/dotfiles`, running alongside
> CodeRabbit. TOOL-013 (#786). This runbook answers the three questions that
> actually get asked: what it costs, what it is made of, and what to do when it
> misbehaves.

## What it is, exactly

No GitHub App or server to run. The Action's workflow and TOML own the runtime
configuration; the push gate, model preflight and audited upstream contract
protect the behavior that depends on PR-Agent internals.

| File | Role |
|---|---|
| `.github/workflows/pr-agent.yml` | when it fires, what credential it gets |
| `.pr_agent.toml` | which model, what it reads, what it must check, what it must never see |
| `scripts/pr-agent-push-gate.sh` | bounded push-review decision (ADR-040) |
| `scripts/pr-agent-model-preflight.sh` | probe the declared NaN chain; pass on the models that answer |
| `scripts/pr-agent-route.sh` | probe the Anthropic member and draw who reviews first from the pool |
| `scripts/pr-agent-publish-guard.sh` | did this run publish a review: the probe before a second attempt, and the final guard |
| `harness/pr-agent-upstream-contract.json` | approved upstream source identities |

The action itself is `The-PR-Agent/pr-agent`, a public **Docker action** pulled
per run — pinned to a commit SHA, never a moving ref. The primary inference
endpoint is NaN's OpenAI-compatible endpoint at `https://api.nan.builders/v1`,
reached with `NAN_API_KEY`. The review pool's Anthropic member (below) reaches
Anthropic's API with `PR_AGENT_ANTHROPIC_API_KEY`. Each attempt receives only its own key.
Both are declared in `secrets/registry.yaml` with
`consumers: ci:mlorentedev/dotfiles`, so `dotf secrets sync ci` manages them.

It is deliberately not a replacement yet: it runs **alongside** CodeRabbit for a
bounded window, because #786 requires recording which tool found what before
either is retired, and the standing rule caps autonomous reviewers at two.

## What it costs, and the limit that actually bites

**One inference call per reviewed event.** It was three; two were turned off as
decisions, and the second of them (#1107) was forced by a measured failure.

| Artifact | Command | State |
|---|---|---|
| the PR body, rewritten with a generated summary | `describe` | **off** — it rewrites the author's body, and the bodies here carry measurement tables worth keeping |
| **PR Reviewer Guide** — the actual review | `review` | **on**. This is the only artifact that attests under GUARD-002 |
| **PR Code Suggestions** | `improve` | **off** (#1107) — `review-attestation.json` excludes its marker because *"suggestions are not a review"*, so it could never turn a PR green while competing for the slot that could |

`/describe` and `/improve` still work as slash commands on an open PR. What
changed is that they no longer run unasked.

### The limit that bites is concurrency, not volume

Measured 2026-08-20 against `api.nan.builders`:

```
8 concurrent -> deepseek-v4-flash :  5 x 200,  3 x 429
2 concurrent -> mimo-v2.5, fired WHILE deepseek was saturated :  2 x 200
```

**Five simultaneous requests per model, and the bucket is per model rather than
per API key.** The pool is shared with pi's TUI, `qq` and hive embeddings, so a
busy session exhausts it — six PRs in one session received no review at all
(#1096, #1100, #1101, #1103, #1104, #1105), every one of them reporting a green
`review` job.

Seven things now stand between that and a silent green:

1. `auto_improve = false` halves what this workflow asks for (#1107).
2. **One review at a time across the repository, none dropped (AI-045 AC10, #1923)**:
   the `review` job joins `pr-agent-review-${{ github.repository }}` with
   `cancel-in-progress: false` and `queue: max`. Without `queue`, GitHub keeps one
   pending run per group and cancels the older one, which is why #1135 removed an
   earlier repository-wide group; `queue: max` keeps up to 100 pending, in order.
   Parallel reviews were measured contending for the same NaN model, so they now
   wait instead. The per-PR workflow group still supersedes a PR's own older
   push. GitHub accepted the job-level key (run 36835280780); that two
   overlapping runs on `main` both complete is the open measurement in AI-045
   `tasks.md`, and a cancelled review run there means the key is ignored: revert
   the block.
3. `fallback_models = ["openai/deepseek-v4-flash", "openai/glm5.3-flash"]`
   behind the primary `openai/mimo-v2.6-flash`: NaN models with their own
   buckets of five, which is what makes them an automatic fallback under
   LiteLLM when the primary is saturated. The draw puts whichever NaN model it
   picked first and keeps the others behind it, and a model that times out
   moves on to the next instead of being retried
   (`CONFIG__RETRY_SAME_MODEL_ON_TIMEOUT: "false"`).
4. **A model preflight (AI-045)**: before PR-Agent starts,
   `scripts/pr-agent-model-preflight.sh` sends one minimal call to each model of
   the declared chain. Only the ones that answer enter the review pool's draw;
   every one that did not is a `::warning::` and a row in the job summary. It exists because NaN retired `mimo-v2.5` on
   2026-09-30: it hung for hours (PR-Agent's fallback does not catch a hang, so
   every PR got a green job and no review), then answered 401.
5. **Streaming (AI-045 AC6, #1858)**: the Action sets
   `LITELLM__CUSTOM_LLM_PROVIDER`, `LITELLM__FORCE_STREAMING_CUSTOM_LLM_PROVIDER`
   and `LITELLM__FORCE_STREAMING_API_BASE_SUBSTRINGS` in its env, so PR-Agent
   streams its NaN calls. A non-streamed answer is cut at the edge after about
   125 s, and a held request sat silent for the step's whole timeout although
   `ai_timeout` was 120 (lesson 327). The pinned PR-Agent build streams only when
   all three are set; re-check that when the pin moves.
6. **A daily canary (AI-045 AC8, #1860)**: `.github/workflows/model-canary.yml`
   runs `dotf harness canary` once a day with no PR involved. It probes every NaN
   model that `harness/model-map.json` and the pin sites in
   `harness/model-pins.json` bind, and keeps one `model-canary` issue current:
   opened or rewritten when a model is refused or stays unavailable, closed by
   the first run in which every model answers. Over-quota answers do not open
   it. Run it locally, where it also reads the deployed pin sites:
   `dotf secrets run --only NAN_API_KEY -- dotf harness canary`.
   Each model is called through the API it serves: the map's `services.rerank`
   and `services.embeddings` answer 404 on `/chat/completions` (measured
   2026-10-01), which a chat-only probe would report as a retirement every day.
   A new non-chat service needs its entry in `serviceAPIs`
   (`cli/internal/nanprobe/bindings.go`); until then its row reads "refused".
7. **A review pool across two providers (AI-045 AC9, #1923)**: see the next
   section.

### The review pool

Members that answered their probe are drawn by the `weight` in their
`pr_agent` block of `harness/reviewer-pool.json` (#1923, amendment B, 2026-10-09),
so each member's share is a declared number:

| Member | Provider | Weight | Declared in |
|---|---|---|---|
| `openai/mimo-v2.6-flash` | NaN | 27 | the pool; `.pr_agent.toml` `model`; the `models` step's `DECLARED_MODEL` |
| `openai/deepseek-v4-flash` | NaN | 19 | the pool; `.pr_agent.toml` `fallback_models`; `DECLARED_FALLBACK_MODELS` |
| `openai/glm5.3-flash` | NaN | 19 | the same (joined 2026-10-08; reviews only at `reasoning_effort: low`, see below) |
| `anthropic/claude-haiku-5-5` | Anthropic | 35 | the pool, effort `high` |
| `anthropic/claude-sonnet-5-5` | Anthropic | risk route | the pool, effort `medium`; never drawn |

Tests hold the pool's NaN models equal to the preflight's chain, and the draw's
share of every member equal to its weight. `scripts/pr-agent-route.sh` draws the
member that reviews first and writes it to the job summary. A member that
refused, hung or was over quota is not in the draw, and its weight is shared out
among the rest for that run.

**The risk route.** A PR whose additions plus deletions reach
`pr_agent_risk.min_changed_lines` (1,500), or that carries the `deep-review`
label, reviews first on Sonnet, with NaN as the second attempt. The route step
reads the size and labels with `gh api`, because an `issue_comment` event
carries neither. When Sonnet does not answer its probe, the PR goes to the draw
and the job summary says why. Add the label to a small PR whose risk is not in
its size (secrets, auth, CI permissions); create it once per repository:

```bash
gh label create deep-review --repo mlorentedev/dotfiles --color 5319e7 \
  --description "Route the PR-Agent review to Sonnet"
```

Every attempt sets `CONFIG__OUTPUT_RUN_DETAILS`, so the published review ends
with the model that wrote it, its tokens and its time.

One attempt talks to one provider, because PR-Agent's fallback chain shares one
transport and NaN's would misroute an Anthropic model. So:

- NaN drawn: the NaN attempt starts on the drawn model, with the other NaN
  models behind it in declared order, and Anthropic is the second attempt.
- Anthropic drawn: Haiku reviews, and NaN is the second attempt.

**glm5.3-flash needs `reasoning_effort: low`.** At its default it reasons until
NaN closes the stream at 60,000 reasoning characters and publishes nothing
(lesson 370). The NaN step sets `CONFIG__REASONING_EFFORT: low` and lists glm in
`CONFIG__ADDITIONAL_REASONING_EFFORT_MODELS`, because PR-Agent sends an effort to
a model LiteLLM does not know only when it is listed. The setting covers the
whole NaN attempt; deepseek accepts it and ignores it, mimo never receives it.

The second attempt runs only when `pr-agent-publish-guard.sh --probe` measures
that nothing was published. The Anthropic step appears twice in the workflow
(first and second), because steps cannot be reordered at run time; a test holds
the copies identical.

**The `PR_AGENT_PROVIDER` repository variable** overrides the draw:

| Value | Effect |
|---|---|
| unset or `draw` | by weight over every member that answered, risky PRs to Sonnet first |
| `nan` | NaN first, Anthropic second, no risk route |
| `anthropic` | Anthropic first, NaN second |
| `nan-only` | NaN only. The Anthropic model is not even probed: **this stops all spending on the key** |

Any other value fails the job: a misspelt `nan-only` must not spend. A forced
provider with no member that answered draws from the rest, with a warning.

```bash
gh variable set PR_AGENT_PROVIDER --repo mlorentedev/dotfiles --body nan-only   # stop spending
gh variable delete PR_AGENT_PROVIDER --repo mlorentedev/dotfiles               # back to the draw
```

**Independence.** For the spec archive gate, whose value is that the reviewer is
not the implementer, `harness/reviewer-pool.json` lets an Anthropic model sign
only beside another vendor's first signature, or as a recorded fallback
(amendment B, 2026-10-09). This pool is a different gate, a first-pass review on
every PR, and admits Haiku and Sonnet as reviewers in their own right by the
owner's decisions of 2026-10-08 and 2026-10-09 (#1923).

### What the Anthropic member may spend

The key is funded from the owner's plan credits, with no card and no
auto-reload, under a spend limit in the Console; the monthly budget is $60-65,
shared with the local adversarial review, which reads the same Bitwarden item as
`REVIEW_ANTHROPIC_API_KEY`. The Console has no per-key model allowlist, so
`ALLOWED_ANTHROPIC` in `scripts/pr-agent-route.sh` is the allowlist (Haiku and
Sonnet): a pool naming any other model fails the draw. `tests/pr-agent-config.bats`
pins it and each of these settings:

| Setting | Value | Why |
|---|---|---|
| `CONFIG__MODEL`, `CONFIG__MODEL_WEAK` | the route's `anthropic_model` | Haiku ($0.10 / $0.50 per MTok up to 100K prompt tokens) when drawn, Sonnet ($2 / $10, no price step) on the risk route. Every model id the toml pins gets an override, or a call to it would leave Anthropic for NaN |
| `CONFIG__FALLBACK_MODELS` | `[]` | no chain: a failure ends the attempt, it never escalates to a pricier model |
| `CONFIG__MAX_MODEL_TOKENS`, `CONFIG__CUSTOM_MODEL_MAX_TOKENS` | `200000` | the NaN attempt's cap, so every member reviews the same diff (owner's choice, 2026-10-08). With Claude's tokenizer counting about 30% more, a diff past ~75K of PR-Agent's tokens crosses the 100K price step and that request pays $0.50 / $2.50 |
| `DEFAULT_ANTHROPIC_CHAT_MAX_TOKENS` | `32000` | LiteLLM's output cap for a Claude model it does not know. Its default, 4096, is shared with thinking and truncated reviews |
| `CONFIG__ENABLE_CLAUDE_ADAPTIVE_THINKING`, `CONFIG__CLAUDE_ADAPTIVE_THINKING_MODELS_OVERRIDE`, `CONFIG__REASONING_EFFORT` | `true`, both allowed models, the route's `anthropic_effort` | adaptive thinking at the member's effort from the pool (Haiku `high`, Sonnet `medium`). PR-Agent's built-in pattern matches neither id, so the override names both; measured with LiteLLM 1.103.0, `thinking` and `output_config` reach the API unchanged |
| `CONFIG__AI_TIMEOUT` | `360` | the call is not streamed, so this bounds the whole answer, thinking included; it ends inside the step's 8 minutes |
| temperature | not sent (`CONFIG__NO_TEMPERATURE_MODELS`, both models) | Haiku 5.5 answers 400 to a non-default temperature, and PR-Agent's default is 0.2 |
| step `timeout-minutes` | `8` | at most two attempts run; the job's 31 minutes cover the worst pair, the probes and the publication measurements |

A worst-case review is about 260K input plus 32K output tokens: roughly $0.21 on
Haiku at its higher prices, and $0.84 on Sonnet. A test holds a price row and a
ceiling for every allowed model ($0.25 and $1.00), so allowlisting a model
without pricing it fails. The budget arithmetic, about $50 a month of the
$60-65, is in `specs/AI-045-nan-catalog-alignment/verification.md`; #2215 and
#2216 measure it against real volume. The spend limit is the backstop, not the
budget.

**Owner setup, once per key rotation:**

```bash
dotf secrets unlock                                   # Bitwarden session
dotf secrets sync ci --repo mlorentedev/dotfiles NAN_API_KEY PR_AGENT_ANTHROPIC_API_KEY
```

Name both secrets. Without names, `sync ci` uploads every var a selected secret
exposes, and `NAN_API_KEY` also exposes `HIVE_WORKER_API_KEY`, a copy no
workflow here reads (#2306).

Until the secret exists, Haiku is out of the draw and the `route` step prints a
`::warning::` naming this command; the final guard repeats it when no review was
published. Check the organization's tier and the Haiku 5.5 and Sonnet 5.5
limits on the Console's *Rate limits* page. The default workspace takes no workspace-level spend or rate
limit; a dedicated workspace is the place for a tighter one.

**Testing each route live** after the sync and the merge (a `/review` comment
runs the default branch's scripts): set `PR_AGENT_PROVIDER` to `anthropic`, then
`nan`, and type `/review` on an open PR each time. Expect the job summary's
"Drawn to review first" to name the forced provider and a Guide published by
that model. Then delete the variable. Before the merge, only a `pull_request`
event on the PR that introduces the pool (its merge ref has the scripts) can
exercise it, for example marking it ready for review.

The multiplier that matters is still the **push**, not the PR. The push gate
requires three newer non-merge commits before another review and requests an
incremental review only with a trusted baseline (ADR-040). The per-PR
concurrency group collapses a burst of pushes into one run.

## Updating the Action without drifting from its contract

The workflow's `uses: The-PR-Agent/pr-agent@<sha>` is the runtime pin. The
upstream filter reads that SHA from the executing workflow at
`github.workflow_sha`; do not maintain a second `PR_AGENT_REF` value.
`tests/pr-agent-config.bats` checks the pinned Action entrypoint, default
configuration and review source files against
`harness/pr-agent-upstream-contract.json`. An update that leaves those files
unchanged needs no manifest edit. If one changes, inspect the upstream diff
against the push gate and publication guard before approving the new blob SHA
in the manifest. A failed API read fails the check; it is not an approval.

`github_action_config.fail_on_tool_errors` is deliberately `true`. The Action
does not identify whether a failed step was caused by the model or a tool, so
the second attempt is never keyed on the first one's `failure` or `cancelled`. It runs only when the publication probe **measures**
that no review was published. An unreadable answer (`unknown`) keeps it off, so
it can never publish a second Guide. The publication guard still fails if
`/review` completes without publishing a Guide. Diagnose a red Action from its
logs; do not re-enable a blind retry that can publish twice.

Every PR-Agent step pins the same `uses:` SHA and repeats the same review
settings. `tests/pr-agent-config.bats` asserts both, so a pin bump moves the
three steps together. When the pin moves, also re-check
`pr_agent/algo/ai_handlers/litellm_ai_handler.py` in the contract: the Anthropic
key path (`ANTHROPIC.KEY`), `no_temperature_models`, and the exclusion of Claude
models from the generic `reasoning_effort` path all live there.

## High-Velocity Batch PR & Triage Workflow

To maximize developer velocity without sacrificing review hygiene:

1. **Sprint Phase (Parallel PR creation)**:
   The developer or agent opens multiple PRs in series for distinct atomic tickets.
   No local pre-push lock blocks PR creation while previous PRs await review.
2. **Async CI Phase (Queued review)**:
   GitHub Actions queues PR-Agent reviews one at a time for this repository.
   Non-slash comments (`## Review triage`, discussions) are
   strictly filtered out (`startsWith(comment.body, '/')`), preventing review loops (#1134).
3. **Triage Sweep Phase (`dotf pr triage-queue`)**:
   Before closing a session or merging, the agent queries `dotf pr triage-queue`.
   The agent evaluates findings, applies fixes via TDD, pushes updates, and posts
   the `## Review triage` table on each PR until the queue reports `[OK] 0 pending`.
   A PR whose diff is exactly a signature under `exempt.signatures` in
   `harness/review-attestation.json` (the release PR) is not listed. The
   attestation gate reports the same diff as exempt; one registry, two
   consumers (#1196).

## Operating it

- **Nothing to trigger.** It fires automatically on the `pull_request` event (`opened`, `synchronize`).
  Slash commands (`/review`, `/describe`, `/improve`) work as comments on an open PR.
- **Attestation under GUARD-002**: PR-Agent posts an issue comment carrying the
  `## PR Reviewer Guide` marker, which `check-review-attestation.sh` recognizes
  as a valid review attestation (`[OK] attested`).
- **Changing the reviewing model** takes two edits that must agree: `model` and
  `fallback_models` in `.pr_agent.toml`, and `DECLARED_MODEL` and
  `DECLARED_FALLBACK_MODELS` in the workflow's preflight step. A test fails when
  they differ. The toml is read from the default branch and the workflow from the
  PR head, which is why both exist. There is one fallback, `openai/deepseek-v4-flash`,
  and the reason is **concurrency, not quality**:
  the limit is per model, so a second NaN model has its own bucket of five.
  What has not changed is the quality bar — `harness/reviewer-pool.json` excludes
  the latency-optimised models (`qwen3.6`, `gemma4`) **by name**, because a
  reviewer that passes cheaply is worse than no gate, and neither may be added
  here. mimo is excluded by neither name nor class; that is a profile match
  (1M context, reasoning) rather than a benchmark, and it is stated as such in
  `.pr_agent.toml`. If a failure survives both models it is still meant to be
  loud, and GUARD-002 plus the no-review guard in the workflow make it so.
- **Changing what it checks**: `[pr_reviewer] extra_instructions`. Every review
  opens with a harness-compliance pass over `AGENTS.md` and `.claude/CLAUDE.md`,
  reported per item even when everything passes.
- **The output is in English** (`response_language = "en-US"`), consistent with
  the English-only durable-record policy for PR comments and code.

## What it must never see

`[ignore] glob` excludes `sensitive/**` before the diff reaches the model. That is
the load-bearing entry: it holds age ciphertext and the DR escrow, and encrypted
or not, credential material is not sent to an inference endpoint to be reviewed.
`tests/pr-agent-config.bats` guards this, along with the model pinning and the
one-secret rule. Read those guards before editing the TOML; they are guards, not
obstacles, but they fail loudly.

## Failure modes seen in production

| Symptom | Cause | Where |
|---|---|---|
| `review` check red, but a Reviewer Guide comment exists | the run was **cancelled**, not failed; `gh pr checks` renders `cancelled` as `fail` | #1040 |
| No review at all, `PR-Agent: skipped` | a comment landed inside the ~20s Docker build window and cancelled the run | #1040 |
| `OPENAI_KEY not set` in the log | benign — the runner prints it while Dynaconf loads `OPENAI__KEY` separately. If the review publishes, the credential is fine | — |
| Comment-triggered runs missing from `gh run list --branch <pr>` | `issue_comment` runs attach to the **default branch**, not the PR | — |
| A `/review` comment on a PR that changes this workflow runs the old steps | an `issue_comment` run executes the **default branch's** workflow file, so it cannot test the PR's version. Toggle the PR to draft and back (`gh pr ready <N> --undo`, then `gh pr ready <N>`): `ready_for_review` runs the PR's own workflow, and the push gate does not hold it back | #2188 |

The first two are one bug with a variable outcome, which is the dangerous shape:
an always-broken reviewer gets noticed, a coin-flip one gets trusted. Both come
from the concurrency group conflating "a new push superseded this" with "someone
commented".

## Related

- `pattern-change-lifecycle.md` — Definition of Done §4 and [[pr-stewardship]]:
  an open PR is not finished work, and its reviewer output is dispositioned.
- [[pr-review-triage]] — the skill that disposes of what a PR came back with.
- `harness/review-attestation.json` — the registry deciding what counts as a
  review having happened. PR-Agent is not in it yet (#1033).
