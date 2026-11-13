---
id: "lesson-594-excluded-from-git-is-not-safe-from-git-clean"
type: lesson
status: active
title: "Excluded from git is not safe from git clean"
created: "2026-11-12"
---

# Excluded from git is not safe from git clean

## Context

A persistent named lane kept its lease, the file that says whether the lane is ACTIVE or PARKED
and when its claim expires, at the worktree root as `.dotf-worktree.json`, listed in
`.git/info/exclude` so it never showed in `git status`. Lesson 272 put it there on purpose: the
liveness signal had to live inside the worktree, where a containerised agent the host cannot see
in `/proc` still reaches it. A `git clean -fdx` inside an active lane, run to reset a build,
deleted it (#3307). `lane status` then read the lane as leaseless, and the pre-commit guard refused
the next commit with a FIX that re-created the state the clean had just destroyed.

## Problem

`info/exclude` and `.gitignore` make a file invisible to `status` and `add`; `git clean -x` exists
to delete exactly those files. A local record that must outlive a reset of the tree cannot live in
the tree, whatever the ignore rules say. The signal of the defect was that the FIX in the guard's
message was itself the recovery for a class, not for a cause: every clean would need it again.

## Solution

Move the lease into the worktree's own git admin dir, `<repo>/.git/worktrees/<id>/`, next to the
identity record ADR-061 already keeps there (ADR-062). Git composes that directory, carries it
through `git worktree move` and removes it with `worktree remove` and `prune`; `git clean` never
reaches it, and a reader resolves it from the `.git` pointer file without a subprocess. The
migration shape: the new writer writes the admin dir and, for one release, the same bytes to the
old location, so a guard a seat has not redeployed and the pinned binary the hooks call keep
reading; readers prefer the admin dir; `lane init` copies a lease it finds only at the old location;
`worktree list` names every worktree still carrying one, which is the precondition for dropping the
mirror. The follow-up that removes the mirror is gated on that report being empty, not on a date.

## Related

- #3307, HARNESS-193 (`specs/HARNESS-193-lease-in-git-dir/`), ADR-061, ADR-062, lesson 272
- `cli/internal/worktree/lease.go`, `git-hooks/lib/lane-guard.sh`
