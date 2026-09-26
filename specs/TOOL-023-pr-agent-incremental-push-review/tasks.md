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

## Closing

- [ ] Every acceptance criterion has a feature with a command that fails without the change.
- [ ] `verification.md` filled; independent review before the archive.
