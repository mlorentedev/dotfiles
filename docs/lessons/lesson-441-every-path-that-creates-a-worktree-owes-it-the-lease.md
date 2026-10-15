---
id: "lesson-441-every-path-that-creates-a-worktree-owes-it-the-lease"
type: lesson
status: active
title: "Every path that creates a worktree owes it the lease, or sweep never sees it"
created: "2026-10-15"
owner: manu
tags: [worktree, lane, lease, sweep, herdr]
---

# Every path that creates a worktree owes it the lease, or sweep never sees it

**Context:** `dotf worktree sweep` reaps a worktree only when it carries `.dotf-worktree.json`: the lease, written by `dotf worktree add`. `dotf lane next` (CLI-119) creates the successor's worktree through `herdr worktree create` instead, because herdr owns the pane the agent runs in (#2547).

**Problem:** Nothing in `internal/lane` wrote the lease, so every lane worktree listed as having no dotf metadata and `sweep` skipped it forever. Measured 2026-10-15: `feat-lane-next`, `feat-worktree-agent` and `review-cli-119` carried `.dotf/lane.json` but no lease, and 10 of 16 dotfiles worktrees had no metadata at all. The lease was a property of one constructor, `worktree.Add`, not of being a worktree, so a second constructor dropped it without any test noticing.

**Solution:** `lane next` leases every worktree it starts an agent in, through the worktree package's own `SaveMetadata`, before it types the agent's command, and keeps a lease it finds so a rerun leaves `created_at` alone. It also leases a worktree another workspace already shows, so a first run that died between create and lease is repaired by the second. A backend answer without a path starts no agent: a worktree the lane cannot lease is one nothing would reap. A test feeds the lane's output to `worktree.ListWithRunner`, so the writer and the reader are held to one file format.

**Why:** When a lifecycle depends on a marker, the marker belongs to every path that creates the thing, not to the command that happened to come first. Enumerate the constructors (`git worktree add`, herdr, `lane next`, a person) and ask of each who writes the marker; any answer of "nobody" is a leak that grows one worktree per run. Write it before the thing is handed to anyone, so a failure leaves something unstarted rather than something running and unowned.
