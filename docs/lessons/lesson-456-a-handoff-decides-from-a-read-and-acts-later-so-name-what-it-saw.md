---
id: "lesson-456-a-handoff-decides-from-a-read-and-acts-later-so-name-what-it-saw"
type: lesson
status: active
title: "A hand-off decides from a read and acts later, so it must name what it saw"
created: "2026-10-16"
owner: manu
tags: [lane, herdr, handoff, cli]
---

# A hand-off decides from a read and acts later, so it must name what it saw

**Context:** `dotf lane next` finds the branch's worktree, creates it when absent and types the successor's command into its pane (CLI-119). Its review found three failures of one shape, ticketed as #2576.

**Problem:** The branch name was the only identity, and "a workspace shows it" was read as "an agent runs in it". Two issues with titles that slug alike shared a branch, so the second was answered "no second agent started" with nothing started. A `Type` that failed after `Create` left a shown workspace with no agent, and every later run gave the same answer, so the queue stalled with no error.

**Solution:** Ask the lease, which the lane itself writes before it types: a worktree leased to another issue is a refusal naming both issues and the retitle that clears it (the new issue's, since the holder's worktree keeps its branch). A shown workspace whose lease is this issue's, older than a grace period, and whose pane has only its shell in the foreground is typed into again. That last test is the process table (`herdr pane process-info`: one foreground process, the shell's own pid), because herdr's own agent field names claude and leaves pi and agy unnamed, and a probe that cannot see an agent types a second one into a working pane. When the probe cannot answer, nothing is typed and the refusal prints the line to run.

**Why:** "Already started" is a claim about the world, and a hand-off that makes it from a name or a display state will make it wrongly in both directions. Decide from a record the lane wrote for exactly this purpose, check liveness with a signal that is the same for every agent, and prefer a stall that names its own fix to a double agent. The remaining race (two runs between `Create` and `Type`) belongs to the atomic claim of CLI-128, not to a lock here.
