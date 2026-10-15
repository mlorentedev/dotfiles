# Run ledger: cli-123-20261015T0747Z

The envelope's `run_id` is empty and the run does not write it back (the operator committed the envelope as is and it is the run's only authority). The id is derived from `armed_at`.

| Item | State | PR | Squash | Reviewer | Notes |
|---|---|---|---|---|---|
| #2547 | landed 2026-10-15T08:2xZ | #2555 | `f0c1b9e5` | PR-Agent (deepseek-v4-flash); CodeRabbit rate-limited | 5 findings: 2 declined, 2 deferred to #2503, 1 done. Lesson 441. Worktree fix-2547-lane-next-lease left in place (operator: no deleting worktrees). |

## Observations

- After #2555 landed, `dotf pr triage-queue` exits 1 on #2556, another session's PR (CLI-118). The run does not triage it. This is the case #2549 fixes.
