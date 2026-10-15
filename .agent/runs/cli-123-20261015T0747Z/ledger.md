# Run ledger: cli-123-20261015T0747Z

The envelope's `run_id` is empty and the run does not write it back (the operator committed the envelope as is and it is the run's only authority). The id is derived from `armed_at`.

| Item | State | PR | Squash | Reviewer | Notes |
|---|---|---|---|---|---|
| #2547 | landed 2026-10-15T08:14:40Z | #2555 | `f0c1b9e5` | PR-Agent (deepseek-v4-flash); CodeRabbit rate-limited | 5 findings: 2 declined, 2 deferred to #2503, 1 done. Lesson 441. Worktree fix-2547-lane-next-lease left in place (operator: no deleting worktrees). |
| #2549 | landed 2026-10-15T08:34:06Z | #2558 | `81d47061` | PR-Agent (deepseek-v4-flash); CodeRabbit rate-limited | 3 findings: 1 done elsewhere (AC3 note on #2546), 2 declined. Lesson 443; runbooks release-dotf, guide-pr-agent-reviewer. AC3 left to the owner (frozen paths). Worktree feat-2549-triage-queue-pr left in place. |
| #2548 | PR open, merge withheld by instruction | #2560 | none | PR-Agent (deepseek-v4-flash); CodeRabbit rate-limited | 1 finding declined (truncated body). ADR-050 `proposed`; ref-form question left open for the owner. Worktree docs-2548-adr-050-lane-contract left in place. |
| #2550–#2554 | parked (not started, not assigned) | none | none | none | One ERROR/WHY/FIX comment each: spec slot (ADR-043 10/10), A free a slot / B `--over-wip-limit`. No `blocked` label exists in the repo, and none was created. |

## Observations

- After #2555 landed, `dotf pr triage-queue` exits 1 on #2556, another session's PR (CLI-118). The run does not triage it. This is the case #2549 fixes.
