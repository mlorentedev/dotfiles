# run 20261106t0654z-bugs — ended queue-empty after 1 iteration / ~0.9h — 0 done, 1 parked, 0 merged, 0 unmerged (this run)

**Waiting on a human, irreversible first:** nothing was landed or published by this run. Two PRs from the session that armed it are open for the owner and were never under this run's grant: they touch harness/, a workflow and ADRs.
- mlorentedev/dotfiles#3212: the PR-Agent re-weight (CI green, triaged).
- mlorentedev/kubelab#3155: its kubelab twin. Both review findings were applied in `35016a1a` and triaged; CI is re-running.

## Why the queue was empty
Five sibling runs were armed in the same minute, and each claims a slice of the dotfiles bug queue in its envelope:
- `chore/autopilot-20261106t0653z`: Windows bugs.
- `…0655z`: doctor, mem, lessons, env, forge and pr land.
- `…plat001c…`: board P1/P2 bugs after PLAT-001c.
- `…cli…`: #3124, #3077, #3114, #3122.
- `…h142…`: review pipeline, worktree, autopilot, #3215, #3220.

Of the open bugs they leave over, none survived the vetoes:

| Issue | Veto |
|---|---|
| #1836 | **parked** (below) |
| #1890 | scope: the zsh and PowerShell fixes live in `install.sh` / `install.ps1`, outside `write_paths`. `install.ps1` is also in the Windows sibling's paths |
| #2634 | 3 a.m. test: narrowing a secret-reading deny rule weakens a security boundary, and the issue leaves the revert to the owner |
| #3133 | `harness/**` is frozen, and the issue is a question for the owner |
| #1547 | needs a design decision (who writes `machine.id`); the doctor check sits in a sibling's paths |
| #1934 | needs the Copilot CLI to reproduce |
| assigned to the owner (#2268, #2102, #1908, …) | leased |

The board query (`gh project item-list`) failed with "unknown owner type" while six runs shared the GraphQL budget, so the queue came from the REST issues list (`labels=bug`).

## Parked, with the question
- **#1836** (HARNESS-167, retired `nan/mimo-v2.5`). dotfiles is clean, but the pool copies in iris, pollex, web and garsync still list the id. **(A)** one PR per repository syncing the copy; **(B)** close as covered by #3211's stale-copy check. Comment: https://github.com/mlorentedev/dotfiles/issues/1836#issuecomment-6679148090. The board Status=Blocked edit failed (GraphQL), so the comment is the only park signal.

## Done but unmerged
None from this run.

## Evidence index
- kubelab#3155 fixes: `poetry run pytest -q --no-cov tests/test_reviewer_pool.py tests/test_pr_agent_route.py`, 76 passed.
- Envelope committed at `f95eac63`; `git diff --exit-code` clean.

## Budget spent
1 of 8 iterations, about 55 minutes, 1 of 6 items. No dollar figure exists for this run.

## Host
Sleep was prevented by caffeinate assertions held by other processes (`pmset -g`). The run ended early, so the window was not needed.

## Next action
Owner: review dotfiles#3212 and kubelab#3155 together, and answer #1836 (A or B).
