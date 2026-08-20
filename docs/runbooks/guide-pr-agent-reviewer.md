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

Three things now stand between that and a silent green:

1. `auto_improve = false` halves what this workflow asks for.
2. A job-level concurrency group serialises inference repo-wide and **queues**
   rather than cancelling — ADR-032's *"queues or escalates, never degrades
   silently"*. Note the ceiling: GitHub holds one pending run per group, so in a
   burst of three or more the extras are cancelled rather than queued. Visibly.
3. `fallback_models = ["openai/mimo-v2.5"]` — a second NaN model with its own
   bucket of five, which is what makes it a fallback rather than a rename.

The multiplier that matters is still the **push**, not the PR: the workflow fires
on every push to the branch, so a PR with five pushes is five reviews of the full
diff. The per-PR concurrency group collapses a burst of pushes into one review of
the settled state.

## Operating it

- **Nothing to trigger.** It fires on the PR event. Slash commands (`/review`,
  `/describe`, `/improve`) work as comments on an open PR.
- **Read the output as a comment, not a review.** PR-Agent posts an *issue
  comment*; it does not submit a formal GitHub review. `gh pr view N --json
  reviews` will not show it. This is why it cannot currently attest under
  GUARD-002 — see #1033.
- **Changing the reviewing model** is one line in `.pr_agent.toml`. There is one
  fallback, `openai/mimo-v2.5`, and the reason is **concurrency, not quality**:
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
