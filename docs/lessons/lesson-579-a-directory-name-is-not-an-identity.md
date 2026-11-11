---
id: "lesson-579-a-directory-name-is-not-an-identity"
type: lesson
status: active
title: "A directory name is not an identity"
created: "2026-11-11"
---

# A directory name is not an identity

## Context

The pre-commit lane guard (HARNESS-186) decided that a worktree was a persistent named lane by comparing its
last path segment with the marker's `canonical_lanes`, and only when the lease file was absent. `dotf lane
init` located lanes the same way, at `<WORKTREE_ROOT>/<repo basename>/<lane>`, and adopted whatever directory
it found there. #3261 reported both halves: a real lane could go unmatched and a same-named stranger was
blocked.

## Problem

Three measurements on a scratch repository (2026-11-11) showed the name answering wrongly in both directions,
and reached by routine commands rather than by hand. `git worktree move` renames a lane and the guard stops
seeing it; `git clean -fdx` deletes the lease, which lives only in `info/exclude`, so the name rule is the one
that fires next (#3307); and a second repository sharing a basename under one `WORKTREE_ROOT` was silently
handed the first repository's lanes by `lane init`, because nothing checked that the directory at the path was
a worktree of the repository being initialised. `HasNamedLanes` answered true for the stranger's lanes for the
same reason, which would route a handoff into them.

## Solution

Ask git where it keeps the worktree's own state and put the identity there: `lane init` writes the lane name to
`dotf-lane` in `git rev-parse --absolute-git-dir`, which git moves with `worktree move`, keeps through `gc` and
`clean -fdx`, and drops with `worktree remove` or `prune` (ADR-061). The guard reads that file and nothing else
for rule 5b; `canonical_lanes` stays in the marker as policy for the CLI. Adoption of an existing directory
now requires common-dir and top-level equality with symlinks resolved, so a plain directory, a main checkout or
another repository's worktree is refused by name and owner. The guard case that pinned the name rule was
rewritten, since the contract it protected was the defect.

Before keying a decision on a path segment, list the commands that change the segment or delete the file the
decision falls back to, and run them: a rule that a rename or a `git clean` can flip is a rule on a coincidence.
When git already owns a per-worktree directory, a record there inherits the worktree's lifetime for free, and
stores no path on any OS.
