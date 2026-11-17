---
id: "lesson-606-a-landing-tools-flag-can-delete-the-worktree-it-was-run-beside"
type: lesson
status: active
title: "A landing tool's flag can delete the worktree it was run beside"
created: "2026-11-15"
---

# A landing tool's flag can delete the worktree it was run beside

## Context

A persistent lane (`feat-lane-secondary`) disappeared twice within seven seconds of `dotf pr land`
merging the PR it held (#3531). The session transcript (UTC) shows `[MERGED] #3526` at 01:53:30 and the
first `lane park` refusing the missing worktree at 01:53:37; then `[MERGED] #3530` at 02:00:29 and the
same refusal at 02:00:36, the directory confirmed gone at 02:00:53. The first reading was that `sweep` or `worktree done` reaped it,
because both can remove a worktree and a squash-merged branch is what they look for.

## Problem

Neither could have. A lane with its lease is parked by both, never removed, and every lane on disk
carried its lease. The code that removes worktrees is a short list, and none of it ran. The removal
was in a binary the code only calls: `gh pr merge --delete-branch` removes the linked worktree that
has the branch checked out, in gh 2.104.0 (the version installed; others were not tested). The flag name says "branch", and the side effect on a
directory is in no help text; it showed only in the strings of the binary ("Removed worktree %s"), and
the source at the v2.104.0 tag confirms it: `pkg/cmd/pr/merge/merge.go` reaches `WorktreeRemove` only
through `deleteLocalBranch`, which returns first unless `deleteBranch` is set, and without the flag a
non-interactive merge sets nothing.

## Solution

Find what ran before reading what could run. The session transcripts held every command of every
agent, and in the seven-second window there was one, `dotf pr land`. That narrowed a search of three
tools to one subprocess, and the binary's strings confirmed the mechanism.

Keep a tool that lands PRs away from worktrees: merge without `--delete-branch`, delete the remote
head over REST, and leave the local branch and worktree to the commands that check the caller's
cwd, running sessions and the lease first. A fork's head is skipped, because the same branch name in
this repository is another branch.

A flag that deletes "the branch" in a tool that knows about worktrees may delete more. Read the
subprocess's own words before trusting its help text, and when a directory goes missing, look for
the command that ran, not only the commands that could.
