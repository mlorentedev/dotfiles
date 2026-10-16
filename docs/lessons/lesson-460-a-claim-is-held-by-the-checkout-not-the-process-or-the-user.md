---
id: "lesson-460-a-claim-is-held-by-the-checkout-not-the-process-or-the-user"
type: lesson
status: active
title: "A claim is held by the checkout, not the process or the user"
created: "2026-10-16"
owner: manu
tags: [lane, claim, concurrency, identity]
---

# A claim is held by the checkout, not the process or the user

**Context:** CLI-128 (#2551) wires the claim of lesson 459 into `lane next` and adds `lane claim` and `lane release`. A claim names its holder, and `release` refuses another holder's claim without `--force`.

**Problem:** Each obvious identity breaks one of the three behaviours the claim needs. `user@host` makes two sessions of the same user one claimant, so the loser of a race reads the winner's claim as its own and both start the lane. `user@host:pid` makes every later invocation another holder: `lane claim` and the `lane release` that follows are different processes, so the owner always needs `--force`. And treating "claimed by me" as a resume would put back the find-then-create race the claim was meant to remove.

**Solution:** The holder is the calling checkout (`user@host:<worktree>`): stable across the invocations of one session, distinct between sessions in different worktrees. A claim held by the same checkout from a run that died before it created the worktree still fails closed, and the refusal names `lane release`. A branch whose worktree already exists is not claimed again, because it was claimed when it was started.

**Why:** Identity in a lock has to match the unit that owns the work and survives as long as the work does. Pick it by asking which of "release by the owner", "two owners never merge" and "a restart is not a second owner" each candidate breaks, and keep the one that breaks none.
