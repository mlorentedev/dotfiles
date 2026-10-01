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

**Two files in this repository, and nothing else.** No GitHub App to install, no
server to run, no integration in another repo, no account beyond the inference
key.

| File | Role |
|---|---|
| `.github/workflows/pr-agent.yml` | when it fires, what credential it gets |
| `.pr_agent.toml` | which model, what it reads, what it must check, what it must never see |

The action itself is `The-PR-Agent/pr-agent`, a public **Docker action** pulled
per run — pinned to a tag, never a moving ref. The only outbound dependency is
NaN's OpenAI-compatible endpoint at `https://api.nan.builders/v1`, reached with a
single secret, `NAN_API_KEY`, declared in `secrets/registry.yaml` with
`consumers: ci:mlorentedev/dotfiles` so `dotf secrets sync ci` manages it.

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

Six things now stand between that and a silent green:

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
3. `fallback_models = ["openai/deepseek-v4-flash"]` behind the primary
   `openai/mimo-v2.6-flash` — a second NaN model with its own bucket of five,
   which is what makes it an automatic fallback under LiteLLM when the primary
   is saturated.
4. **A model preflight (AI-045)**: before PR-Agent starts,
   `scripts/pr-agent-model-preflight.sh` sends one minimal call to each model of
   the declared chain. The review runs on the first one that answers; every one
   that did not is a `::warning::` and a row in the job summary, and the job
   fails when none answers. It exists because NaN retired `mimo-v2.5` on
   2026-09-30: it hung for hours (PR-Agent's fallback does not catch a hang, so
   every PR got a green job and no review), then answered 401.
5. **Streaming (AI-045 AC6, #1858)**: every attempt sets
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

The multiplier that matters is still the **push**, not the PR: the workflow fires
on every push to the branch, so a PR with five pushes is five reviews of the full
diff. The per-PR concurrency group collapses a burst of pushes into one review of
the settled state.

## High-Velocity Batch PR & Triage Workflow

To maximize developer velocity without sacrificing review hygiene:

1. **Sprint Phase (Parallel PR creation)**:
   The developer or agent opens multiple PRs in series for distinct atomic tickets.
   No local pre-push lock blocks PR creation while previous PRs await review.
2. **Async CI Phase (Parallel review)**:
   GitHub Actions and PR-Agent review each PR in parallel in the cloud across
   available NaN slots. Non-slash comments (`## Review triage`, discussions) are
   strictly filtered out (`startsWith(comment.body, '/')`), preventing review loops (#1134).
3. **Triage Sweep Phase (`dotf pr triage-queue`)**:
   Before closing a session or merging, the agent queries `dotf pr triage-queue`.
   The agent evaluates findings, applies fixes via TDD, pushes updates, and posts
   the `## Review triage` table on each PR until the queue reports `[OK] 0 pending`.

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
- **The output is in Spanish** (`response_language = "es-ES"`). The instructions
  themselves stay in English, as does everything the English-only standing order
  names — commits, branch names, PR titles and bodies, code comments.

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
