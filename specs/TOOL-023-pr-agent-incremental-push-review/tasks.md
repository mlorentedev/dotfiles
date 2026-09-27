---
tags: [spec, tasks, templates]
created: "2026-09-25"
---

# Tasks - TOOL-023-pr-agent-incremental-push-review

> TDD order. One task = one focused commit. `[AC<n>]` = serves acceptance criterion n.

## Setup

- [x] Worktree `../dotfiles-wt-pr-agent-incremental`, branch `feat/pr-agent-incremental-push-review` from main
- [x] #1756 on the board, In Progress, P1

## Implementation

- [ ] [AC1] Failing cases in `tests/pr-agent-push-gate.bats`, on JSON fixtures with the real `jq`:
  - below the threshold, at it, and past it;
  - merge commits do not count;
  - the previous review is found by identity line or heading prefix, and a later human comment is not one;
  - a bot push is skipped;
  - with no previous review, the gate runs;
  - on an unreadable or missing input, the gate runs;
  - a usage error exits 2.
- [ ] [AC1] `scripts/pr-agent-push-gate.sh` makes them pass.
- [ ] [AC2] Failing workflow cases in `tests/pr-agent-config.bats`:
  - the gate runs only on `synchronize`;
  - PR-Agent and the guard are skipped on `run == 'false'`;
  - `push_commands` is `["/review -i"]`.
- [ ] [AC2] Wire the gate into `.github/workflows/pr-agent.yml`, with a sparse checkout of the script.
- [ ] [AC3] Failing cases: the registry declares `## Incremental PR Reviewer Guide`, and the guard reads every marker. Then make them pass.
- [ ] [AC4] Replace the #1053 rationale. Write ADR-040. Add the draft sentence to `pr-stewardship` (vault, then render).

## Follow-up (#1757 triage, comment 5851151835)

- [x] [f5] Failing cases for a comment carrying a review marker from a non-bot login, with and without a real `github-actions[bot]` baseline present; author-filter the baseline jq, and treat a newer forged marker as `mode=full` (CWE-345).
- [x] [f6] Failing cases: a `head_sha` in the state block counts commits by position (immune to a rebase preserving author dates); the sha absent from the PR's commits forces `mode=full`; a state block without a `head_sha` (every incremental review; PR-Agent's `_review_finding_state_enabled` disables it) falls back to the date count; unreadable commits on the sha path still fail open.
- [x] [f7] Confirm the workflow can select per push (an `env:` value may read a step output, as `STARTED`/`HEAD_SHA` already do); gate emits `mode=full|incremental`, `pr-agent.yml`'s `push_commands` reads it; update `tests/pr-agent-config.bats`'s exact-equality check for the template.
- [x] Read the pinned PR-Agent v0.45.0 source to confirm incremental reviews carry no state block, rather than assume it.
- [x] Disclose a fourth, narrower gap surfaced by reviewing this fix: SHA-position counting can disagree with PR-Agent's own date-based `get_commit_range` on a cherry-pick with a preserved old author date. Amended ADR-040 D2 rather than leaving it as an absolute guarantee; the pre-existing "no review published" guard (#1107) is what actually catches it.
- [ ] File the upstream finding: `get_previous_review` picks a baseline by comment body only, with no author check, so a forged comment can become PR-Agent's own incremental baseline too (not filed by this agent; owner's call per the triage).

## Closing

- [ ] Every acceptance criterion has a feature with a command that fails without the change.
- [ ] `verification.md` filled; independent review before the archive.
