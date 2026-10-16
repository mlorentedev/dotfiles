---
id: "lesson-455-an-empty-key-matches-the-newest-row-and-a-head-must-be-ready-to-land"
type: lesson
status: active
title: "An empty lookup key matches the newest row, and a queue head must be ready to land"
created: "2026-10-16"
owner: manu
tags: [lane, reconcile, forge, testing]
---

# An empty lookup key matches the newest row, and a queue head must be ready to land

**Context:** `dotf lane plan` names the lowest-numbered open Review lane as the landing head and, from #2596, evaluates that PR's landing conditions through `prland.Decide` (CLI-127, #2550). The first run against the real forge, not a fixture, found two defects every unit test had passed.

**Problem:** (1) A detached-HEAD worktree has an empty branch. `REST.PRFor("")` queried the pulls endpoint with `head=<owner>:` and the forge answered with the newest PR, so one detached worktree was attributed PR #2601, three times over. A fixture forge keyed by branch name never receives an empty key. (2) A draft PR (#2582) was the lowest-numbered open PR and became the landing head, so the plan's one command was `land` for a PR the grant refuses to land.

**Solution:** `PRFor("")` returns "no PR" without making a request, and the REST row carries `draft`, which `markHead` skips. `plan` now asks `prland.Blockers` for the head's conditions, so a head that cannot land says why instead of printing a command that will fail.

**Why:** A lookup by a key that can be empty needs an explicit answer for the empty key; the transport's default (a filter that matches everything) is the wrong one. And "first in the queue" is a position, not a readiness: whichever rule picks the head must use the same predicate as the verb that acts on it, the same shape as lesson 454. Both were visible only to a run over live data, so the closing check of a reconcile feature is one read-only `plan` against the real forge, with the output read.
