---
id: "ADR-037-review-runners-bounded-and-budgeted"
type: adr
status: accepted
owner: manu
date: "2026-09-25"
supersedes: []
extends: [adr-032-cross-harness-agent-orchestration]
issue: mlorentedev/dotfiles#1710
tags: [architecture, decision, sdd, review, agents, timeout]
created: "2026-09-25"
---

# ADR-037: Every independent review runs under a deadline and knows its time budget

## Context

`dotf spec review --timeout` promised that "a stuck run should be noticed, not waited on", but only agy was bound, through its own `--print-timeout`. pi, the runner most of the reviewer pool uses, ran for as long as it ran: SKILL-001's review rounds took 31, 39 and 70 minutes.

A deadline alone would have killed the slow rounds before they wrote a verdict, which loses the whole run. And the runner is a grandchild of the launcher (`dotf secrets run -- pi …`), so killing the launcher's direct child would have left pi running and spending.

## Decision

1. **Every runner runs under `dotf spec deadline <timeout> -- <runner>`**, in tmux and in the foreground. The launcher wraps the runner; no runner's own flag is relied on.
   - The deadline stops the runner's whole process group: SIGTERM, then SIGKILL after 10 seconds.
   - It forwards SIGINT, SIGTERM and SIGHUP to that group.
   - A stopped run exits 124 and says that the time limit, not a verdict, ended it.
2. **The reviewer's prompt states its budget:** the stop time, and a target at two thirds of it, both as absolute clock times the model can check with `date`. From the target on, it writes what it has verified and marks the rest UNVERIFIED.
3. **The default deadline is 45 minutes, a 30-minute target.** It is a policy value measured from real runs. Change it with data (the `review-request.json` and `review.md` times of recent reviews), not by feel.

## Consequences

- The first review under this ADR was told to aim for 16:30 and would have been stopped at 16:45. It wrote its verdict at 16:17, 17 minutes, and ran its own mutation checks. The next review, MEMORY-009's, took 10 minutes.
- A runner added to the pool later is bound without code of its own. `dotf agent` dispatch already requires a timeout (ADR-032 §2), so every agent run the harness launches is now bounded.
- On Windows, process groups do not carry signals, so the deadline kills the runner but not its children. Reviews there run in the foreground, in the operator's terminal.
- Lessons 298 (bound a wrapped runner by its process group) and 299 (tell the reviewer its budget) record the mechanics. The implementation is #1721.
