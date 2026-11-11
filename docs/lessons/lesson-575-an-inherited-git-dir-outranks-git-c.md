---
id: "lesson-575-an-inherited-git-dir-outranks-git-c"
type: lesson
status: active
title: "An inherited GIT_DIR outranks git -C"
created: "2026-11-10"
---

# An inherited GIT_DIR outranks git -C

## Context

`dotf worktree` runs every git command as `git -C <repo> …`, and was assumed to act on the repository it names.
#3122 found the listing reading another repository when a hook had exported `GIT_DIR`; #3275 found the same for
`branch -D`, `worktree remove` and the claim-ref pushes in `done` and `sweep`.

## Problem

`-C` only changes git's working directory, and repository discovery from the working directory happens only
when `GIT_DIR` is unset. git exports `GIT_DIR` (and sometimes `GIT_WORK_TREE`, `GIT_INDEX_FILE`) to hooks and
to commands it runs, so a `dotf` launched from a hook, or a test run under one, inherits them. Measured in
`TestRealSweepRunner_DeletesTheBranchOfTheNamedRepositoryWhateverGitDirSays`: with `GIT_DIR` naming another
repository, `BranchDelete(mine, "doomed")` deleted the other repository's `doomed` and left ours in place.
Rebuilding a command's environment from `os.Environ()` to add one variable (`GIT_TERMINAL_PROMPT=0`,
`LC_ALL=C`) brings the inherited variables back, which is how the first fix missed the fetch.

## Solution

Every git call in `cli/internal/worktree` goes through `gitIn(dir, args...)`, which drops the
repository-location variables; a call that needs one more variable appends to `cmd.Env`, never to
`os.Environ()`. `TestPackage_RunsGitOnlyThroughGitIn` parses the package and fails on any
`exec.Command("git", …)` or `os.Environ()` outside `gitIn`. Any Go code that names a repository with `-C`
needs the same scrub; doctor keeps its own copy of the list, merged under #3232.
