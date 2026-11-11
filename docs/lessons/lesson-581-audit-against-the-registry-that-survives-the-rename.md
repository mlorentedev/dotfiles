---
id: "lesson-581-audit-against-the-registry-that-survives-the-rename"
type: lesson
status: active
title: "Audit against the registry that survives the rename"
created: "2026-11-11"
---

# Audit against the registry that survives the rename

## Context

#3262 asked for a check that reports drift between `.dotf/lanes.json` and the lanes on disk in both
directions, with "a renamed lane directory is reported as drift" as acceptance. The obvious
enumeration is the one `lane init` uses: look at `<WORKTREE_ROOT>/<repo>/<lane>` for each declared
lane, and list the repository's worktrees with `git worktree list` for the other direction.

## Problem

Measured on a scratch repository (2026-11-11) before writing the check: after `mv feat-lane-primary
renamed`, the lane's canonical path is gone, `git worktree list` still names the old path and marks it
`prunable`, and nothing lists the new one. A path-based audit would report the lane absent and
recommend `dotf lane init`, which fails: `git worktree add` at a path git has registered answers
"missing but already registered worktree". The FIX line would send the operator to the one command
that cannot repair the state. What survived the rename was `<repo>/.git/worktrees/<id>/`: its
`gitdir` file still pointed at the old path and the `dotf-lane` record was intact, which is the
whole point of putting the identity there (ADR-061).

## Solution

`lane.AuditLanes` enumerates git's own registry of linked worktrees, `<common dir>/worktrees/*/`,
reads each record and `gitdir`, and compares that with the marker. A record whose registered path no
longer exists is "moved or deleted by hand" with `git worktree repair <new path>` or `git worktree
prune` as the FIX; a record at a path other than the canonical one is "moved" with `git worktree move`
back; only a lane with no record at all is "absent" with `dotf lane init`. Before writing a check that
reports absence, run the commands that produce the absence and read what the system kept: the
registry that survives the rename is where the audit reads, and the FIX line follows from which
state the operator is actually in.
