---
id: "TOOL-013-pr-agent-nan-review"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-06"
tags: [spec, proposal]
template_version: "1.0"
---

# TOOL-013-pr-agent-nan-review

> Work-gate: `mlorentedev/dotfiles` issue **#786** (TOOL-013, board `Backlog`, P2).
> Prior art: `knowledge` vault `10_projects/dotfiles/research/research-pr-review-agents.md`
> (2026-06-11, 12-tool survey) and `10_projects/ai-strategy/adversarial-review-2026-08.md` §9.

## Why

The free CodeRabbit tier gives line-by-line reviews **only on public repositories**. Of the ten
active personal repos, three are private — `resume`, `biwenger-mvp`, `knowledge` — and receive
walkthrough summaries only, permanently. That gap is not a pricing accident we can wait out; it is
the plan's design. It has already produced a concrete miss: a merge burst of seven PRs in about
forty minutes exhausted CodeRabbit's per-developer rate limit, and six of them received no
walkthrough at all.

The June survey ranked **PR-Agent** (`qodo-ai/pr-agent`, MIT) as the bring-your-own-key fallback and
rejected it on one line: *"local-model quality drops below the bar next to Codex"*. That judgement
evaluated BYO as "Ollama on homelab hardware". Cluster inference is a different quality tier
entirely, so the premise behind the rejection no longer holds. The project is also actively
maintained — last commit 2026-08-06, ~12.4k stars, not archived.

This spec adopts PR-Agent as a **reusable workflow owned by this repo**, backed by NaN inference,
applied uniformly to every personal repo.

## What

After this lands and is rolled out:

- `.github/workflows/pr-review.yml` in this repo is a `workflow_call` reusable workflow that runs
  PR-Agent against an OpenAI-compatible endpoint.
- Each participating repo carries a short caller workflow (target: under 15 lines) and inherits
  every default from here — model routing, ignore globs, permissions, runner choice.
- Review comments are posted inline on PRs in repos where CodeRabbit cannot reach.
- Sensitive material is excluded **by path glob**, in shared config, so the exclusion is enforced by
  a file rather than remembered by a human.

### Where review quality actually comes from

Recorded here because it drives the configuration priorities below, and because the instinct is to
reach for the prompt first — which is the smallest lever of the four.

1. **Context assembly — dominant.** A reviewer that sees only the changed hunk cannot find a
   cross-file defect. No prompt recovers information that never entered the context window.
   PR-Agent's patch-extension and file-prioritisation logic is precisely the part worth adopting
   rather than reimplementing.
2. **Model class.** At equal context, a large reasoning model finds defects a small one cannot.
3. **Static analysis fed into the prompt.** A large share of what reads as "AI quality" in hosted
   reviewers is linter and SAST output rendered as prose. **This is mostly redundant for us**:
   ruff, mypy strict, BATS, hadolint, gitleaks and per-repo validators already run in CI. Our
   marginal gap is reasoning-level defects, not lint.
4. **Prompt and output format.** Real, but it buys noise control, severity thresholds and
   formatting — it does not create findings.

Consequence: configure context and models carefully; treat prompt tuning as the last pass, run
against real reviews rather than in the abstract.

### Free win: AGENTS.md enters the prompt

`config.repo_context_files` defaults to `["AGENTS.md"]`, read from the default branch. Every repo
here already carries `AGENTS.md` as its behavioural SSOT, so standing orders, the English-only
artifact rule and the no-auto-merge rule become review criteria at zero cost. This is a strong
argument for PR-Agent over any hosted reviewer, which cannot see that file as instruction.

### Model routing — three models, three jobs

```toml
[config]
model             = "deepseek-v4-flash"   # reasoning + long context for large diffs
fallback_models   = ["qwen3.6"]           # independent failure mode — see below
model_weak        = "gemma4"              # descriptions, commit summaries, cheap passes
response_language = "en-US"
```

**Why not `deepseek → mimo → qwen3.6`.** `deepseek-v4-flash` and `mimo-v2.5` draw on the *same*
metered monthly pool. Falling back from one to the other does not survive pool exhaustion — the
exact failure mode a fallback exists for. `qwen3.6` is documented as rate-limited only, which makes
it the genuinely independent lane. `mimo` contributes nothing here: it is omnimodal and the payload
is text.

**Quota is not the binding constraint.** A 500-line diff is roughly 20K tokens. Even at a hundred
reviews a month with multiple passes this stays far below the pool.

> **Re-verify before pinning.** These model names and quota classes come from the vault's
> `21-nan-cloud.md`, verified June 2026. Ticket **AI-001** (`mlorentedev/knowledge` #150) refreshes
> that reference. If AI-001 has landed by implementation time, reconcile against it first — the
> reported `glm5.2` may be the better primary, and the metered/unmetered split may have changed.
> The *shape* of the routing (reasoning primary, independent-lane fallback, weak model for cheap
> tasks) survives any renaming; the literal names may not.

**LiteLLM registry note.** PR-Agent resolves models through LiteLLM, which will not know NaN's
models. `custom_model_max_tokens` must be set explicitly, and `max_model_tokens` (default 32000)
raised if long-context review is wanted.

### Uniform rollout, exclusion by path

```toml
[ignore]
glob = ["50_work/**", "10_projects/fae-brain/**", "10_projects/openkm-brain/**"]
```

The diff leaves our infrastructure only at the model call, and path filters run before it. This is
strictly tighter than the CodeRabbit App, which holds standing read access to every repo in scope;
PR-Agent sees one diff, once, with excluded paths already removed.

`knowledge` has had eight PRs in its lifetime and none since 2026-06-20 — it is not PR-driven, since
obsidian-git commits straight to master. The glob therefore covers the rare bulk-operation case, not
daily flow.

### Runner split, and why it is a security decision

| Repo class | Runner | Reason |
|---|---|---|
| Public (7) | GitHub-hosted | Actions minutes are free on public repos. **Self-hosted must not be used here**: a fork PR would execute on the Beelink |
| Private (3) | Self-hosted, or hosted minutes | No fork-PR exposure; the self-hosted runner avoids consuming the private-repo minute allowance |

On fork PRs the `pull_request` event does not expose secrets, so `NAN_API_KEY` is absent and
PR-Agent degrades to a clean no-op rather than failing loudly. That is the desired behaviour and
requires no extra handling.

### Why a reusable workflow reaches private repos

`workflow_call` from another repository requires the *called* workflow's repo to be public (or
same-org with sharing enabled). This repo is public, so private callers can reference it. This is
load-bearing — the whole one-place-to-change design depends on it.

### Secret distribution

This is a personal account, so organisation-level secrets do not exist; `NAN_API_KEY` must be set
per repository. Per the no-manual-fixes rule this is scripted, not clicked ten times:

```bash
for r in dotfiles kubelab web hive ts-bridge iris pdf-modifier-mcp resume biwenger-mvp knowledge; do
  gh secret set NAN_API_KEY -R "mlorentedev/$r" --body "$NAN_API_KEY"
done
```

Rotation follows the same loop. The key is never written to a repo file and never echoed in logs.

### Reference caller (design sketch, not shipped by this spec)

```yaml
name: PR review
on:
  pull_request:
    types: [opened, synchronize]
  issue_comment:
    types: [created]
jobs:
  review:
    uses: mlorentedev/dotfiles/.github/workflows/pr-review.yml@main
    permissions:
      pull-requests: write
      contents: read
      issues: write
    secrets:
      NAN_API_KEY: ${{ secrets.NAN_API_KEY }}
```

## Out of scope

- **The workflow files themselves.** This spec's diff is the three files under
  `specs/TOOL-013-pr-agent-nan-review/`. The YAML and TOML above are design blocks, to be
  implemented in the follow-up PR this spec gates.
- **Retiring CodeRabbit.** Deliberately deferred — see below.
- **Work-IP repositories.** `fae-brain` and `openkm-brain` are Copilot-covered and IP-restricted;
  they are excluded by path glob and are not rollout targets.
- Prompt/`extra_instructions` tuning beyond defaults. It is the last pass and needs real reviews to
  tune against.
- Any change to CI quality gates. PR-Agent is advisory and must never become a merge blocker.

## Do not retire CodeRabbit on day one

CodeRabbit costs nothing on the seven public repos and carries the highest measured recall of the
twelve tools surveyed. Trading that for uniformity, in advance and without evidence, is a bad deal.

The standing rule caps autonomous reviewers at **two** (documented failure mode: stacking three or
four produces 60+ comments per PR). Codex and CodeRabbit currently occupy both slots. Therefore:

- Run PR-Agent alongside CodeRabbit on public repos for a **bounded** window — roughly three weeks
  or ten PRs.
- Record what each tool found, what only one found, and how much noise each produced.
- Retire one on that evidence, restoring the cap at two.

The overlap is a deliberate, time-boxed exception to the two-reviewer rule, not an oversight.

## Risks / open questions

- **Model names may be stale.** Resolved by design: routing is specified by *role* and re-verified
  against `21-nan-cloud.md` after AI-001 lands, before pinning.
- **NaN has no SLA** (community cluster). Accepted: review is advisory. A provider outage must
  degrade the review and never block a PR — enforced by `continue-on-error` on the review step.
- **LiteLLM may not resolve NaN model names.** Mitigated by `custom_model_max_tokens`; a smoke call
  against the endpoint is the first implementation task, before any workflow is written.
- **Fork-PR execution on self-hosted runners.** Resolved by design: public repos use
  GitHub-hosted runners exclusively.
- **Noise.** Real and the most likely reason this gets abandoned. Mitigated by piloting on one repo,
  and by deferring prompt tuning until there is output to tune against.
- **Open question — `resume` PII.** `data/personal.yml` carries personal data. Sending those diffs
  to a community cluster is the maintainer's call; it is personal data, not corporate IP, so it is
  a judgement rather than a rule. Default in this spec: **include** `resume`, and add a path glob
  for `data/personal.yml` if the answer is no.
- **Open question — token accounting.** There is no telemetry for NaN usage outside the hermes
  microVM, so PR-review consumption will be invisible until that gap is closed.

## Acceptance criteria

1. A reusable `workflow_call` workflow exists in this repo and is referenced by at least one caller.
2. A PR in the pilot repo receives inline review comments produced through the NaN endpoint.
3. A file matching an ignore glob is demonstrably absent from what is sent to the model.
4. A forced provider failure produces a skipped or failed review step while the PR remains mergeable.
5. `NAN_API_KEY` is set on every participating repo by the scripted loop, and appears in no log.
6. Public-repo callers run on GitHub-hosted runners; no public-repo workflow targets `self-hosted`.
