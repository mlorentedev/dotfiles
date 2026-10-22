---
id: "lesson-511-a-claim-every-session-makes-as-one-user-excludes-no-session"
type: lesson
status: active
title: "A claim every session makes as one user excludes no session"
created: "2026-10-21"
---

# A claim every session makes as one user excludes no session

## Context

Before work starts, a session claims an issue in two ways: it assigns the issue to itself and comments "Picked up.
Branch: …". `dotf worktree add --issue N` also records N in the new worktree's lease file.

## Problem

On 2026-10-21 two sessions on one machine fixed #2824 in parallel. One claimed it at 14:40Z; the other opened #2861
for it at 14:44Z, and #2861 merged first, so #2860's work was thrown away. Every session runs as the same GitHub
user, so the assignee was already "taken" by both, and nothing reads the comment before it starts. The lease named
the issue, but `add` never looked at other leases.

## Solution

`add --issue N` lists the repository's worktrees and refuses while another one holds a live lease on N, naming its
path and branch (#2877). An expired lease does not block, and `--share-issue` adds a second worktree on purpose. A
failed listing refuses too, naming the override. This covers one machine only; a claim across machines needs a
forge-side signal.

## Rule

A claim excludes others only if it carries something that tells claimants apart and something reads it before
work begins. When every actor shares one identity, put the claim where each actor is distinct (here, the worktree
on disk) and check it at the point where work is created.
