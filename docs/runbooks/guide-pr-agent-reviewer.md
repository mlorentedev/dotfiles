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

## What it costs — read this before assuming it is one call

**It consumes the NaN subscription on every pull-request event, three times.**

A clean run publishes three artifacts, measured on #1042:

| Artifact | Command | Model |
|---|---|---|
| the PR body, rewritten with a generated summary | `describe` | `model_weak` — qwen3.6 |
| **PR Reviewer Guide** — the actual review | `review` | `model` — deepseek-v4-flash |
| **PR Code Suggestions** | `improve` | `model` — deepseek-v4-flash |

The workflow passes no `auto_describe` / `auto_review` / `auto_improve` inputs, so
**all three run by default**. Diff size measured on #1037: 20 412 tokens against a
stated 200 000 budget, so a review reads the whole diff rather than a window.

The multiplier that matters is not the PR, it is the **push**. The workflow fires
on `opened`, `synchronize`, `reopened` and `ready_for_review` — `synchronize` is
every push to the branch. A PR with five pushes is roughly fifteen inference
calls, on the full diff each time.

### The lever, if consumption needs cutting

Turning off the two passes that are not the review cuts it to about a third:

```toml
[github_action_config]
auto_describe = false
auto_improve  = false
auto_review   = true
```

Worth considering on its own merits, not only for cost: `describe` **rewrites the
PR body**, wrapping the author's text under a `### User description` heading and
adding a generated summary above it. If the PR body is written carefully, that is
a change you may not want.

## Operating it

- **Nothing to trigger.** It fires on the PR event. Slash commands (`/review`,
  `/describe`, `/improve`) work as comments on an open PR.
- **Read the output as a comment, not a review.** PR-Agent posts an *issue
  comment*; it does not submit a formal GitHub review. `gh pr view N --json
  reviews` will not show it. This is why it cannot currently attest under
  GUARD-002 — see #1033.
- **Changing the reviewing model** is one line in `.pr_agent.toml`. There is no
  fallback model, by decision: `harness/reviewer-pool.json` excludes the
  latency-optimised models by name, and a reviewer that passes cheaply is worse
  than no gate. A failure is meant to be loud, and GUARD-002 makes it so.
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
