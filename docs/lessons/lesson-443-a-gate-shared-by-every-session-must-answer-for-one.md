---
id: "lesson-443-a-gate-shared-by-every-session-must-answer-for-one"
type: lesson
status: active
title: "A gate every session runs must be able to answer for one session's change"
created: "2026-10-15"
owner: manu
tags: [pr, triage, gate, parallel-sessions, landing]
---

# A gate every session runs must be able to answer for one session's change

**Context:** `dotf pr triage-queue` lists the open PRs whose reviewer output has no `## Review triage` after it, and exits 1 when the list is not empty. Every lane runs it before `dotf pr land N` (#2549).

**Problem:** The queue only answered for the whole repo, so `triage-queue && pr land N` stopped on any session's untriaged PR. On 2026-10-15 #2542 held up #2539, which was clean. This run measured the same thing within an hour: right after #2555 landed, the repo-wide queue exited 1 on #2556, then on #2557, both other sessions' PRs, while the PR about to land had nothing pending. `release-dotf.md` worked around it in prose ("merge if this PR is not in the list"), which turned a gate into something an agent has to read and interpret.

**Solution:** `--pr N` asks about one pull request only. It reads `pulls/N` and N's comments and never the open-PR list, and it goes through the same verdict path as the repo-wide queue (`pendingOf`). It keeps that queue's refusals: a PR that cannot be read is an error, and so is one that is not open, because an empty answer for a merged PR would read as clear to land. A test asserts the list endpoint is never requested. Live: `--pr 2556` exited 0 while the repo-wide queue exited 1 on #2557.

**Why:** When a gate is shared, its scope decides who can block whom. If a check can only answer for everything, any one participant's unfinished work stops all the others, and people learn to read around it, which is the first step to ignoring it. Give the check the same scope as the action it guards (landing PR N needs N's verdict), and leave the wide form for the questions that really are about everything, such as a session-start brief.
