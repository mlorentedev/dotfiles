---
tags: [spec, tasks]
created: "2026-08-06"
---

# Tasks - TOOL-013-pr-agent-nan-review

> One task = one focused commit. Tick as you go.
> **Status: draft.** Nothing below is started — this spec is written ahead of implementation.

## Setup

- [ ] Branch (worktree) created from main for the implementation PR
- [ ] `proposal.md` complete; acceptance criteria testable
- [ ] Re-verify NaN model names and quota classes against `21-nan-cloud.md` — reconcile if AI-001
      (`mlorentedev/knowledge` #150) has landed. Do not pin names taken from the June snapshot

## Pre-flight (before writing any workflow)

- [ ] Smoke the endpoint directly: one `curl` to `https://api.nan.builders/v1/chat/completions`
      with the chosen primary model, confirm a 200 and a sane completion
- [ ] Confirm LiteLLM resolves the model string PR-Agent will send (`openai/<model>` + `api_base`);
      record the exact working string
- [ ] Determine `custom_model_max_tokens` for each of the three models
- [ ] Decide whether to raise `max_model_tokens` above the 32000 default

## Secret distribution

- [ ] Set `NAN_API_KEY` on every participating repo via the scripted loop in `proposal.md`
- [ ] Verify presence without printing the value (`gh secret list -R …`)
- [ ] Record the rotation procedure in the repo's secrets documentation

## Reusable workflow (this repo)

- [ ] `.github/workflows/pr-review.yml` with `on: workflow_call`, declaring the `NAN_API_KEY` secret
- [ ] Pin the PR-Agent action to a tag, not a moving ref
- [ ] Permissions minimal and explicit: `pull-requests: write`, `contents: read`, `issues: write`
- [ ] `continue-on-error: true` on the review step — provider failure must never block a PR
- [ ] Runner selectable by caller input, defaulting to GitHub-hosted
- [ ] Guard: fail fast with a clear message if `NAN_API_KEY` is unset on a non-fork event
- [ ] `actionlint` clean

## Shared configuration

- [ ] `.pr_agent.toml` defaults: model routing, `response_language`, ignore globs
- [ ] Confirm `repo_context_files` picks up `AGENTS.md` from the default branch
- [ ] Document how a repo overrides a default locally without forking the workflow

## Pilot — biwenger-mvp

- [ ] Add the caller workflow
- [ ] Open a PR containing a **known, deliberate defect**; confirm PR-Agent reports it
- [ ] Confirm inline comments anchor to the right lines
- [ ] Confirm an ignored path is absent from the model payload (verbose log inspection)
- [ ] Force a provider failure (bad key); confirm the step degrades and the PR stays mergeable
- [ ] Exercise the slash commands: `/review`, `/describe`, `/improve`, `/ask`

## Rollout

- [ ] Public repos on GitHub-hosted runners: kubelab · web · hive · dotfiles · ts-bridge · iris ·
      pdf-modifier-mcp
- [ ] Private repos: resume (subject to the PII open question) · knowledge (path-glob protected)
- [ ] Confirm no public-repo caller targets `self-hosted`
- [ ] Confirm a fork PR produces a clean no-op rather than a failure

## Evaluation window (bounded, ~3 weeks / ~10 PRs)

- [ ] Record per PR: findings by tool, unique findings, false positives, comment volume
- [ ] Decide which of PR-Agent / CodeRabbit to retire; restore the two-reviewer cap
- [ ] Capture the outcome as a lesson in `docs/lessons.md`

## Closing

- [ ] Each acceptance criterion covered by evidence in `verification.md`
- [ ] Lint passes; no unrelated changes in the diff
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] Update the vault research note — it currently rejects PR-Agent on a premise this spec refutes
- [ ] `/spec archive TOOL-013-pr-agent-nan-review`
