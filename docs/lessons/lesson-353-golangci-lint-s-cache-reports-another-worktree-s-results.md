---
id: "lesson-353-golangci-lint-s-cache-reports-another-worktree-s-results"
type: lesson
status: active
title: "golangci-lint's cache reports another worktree's results"
created: "2026-10-07"
---

# golangci-lint's cache reports another worktree's results

## Context
Several worktrees of this repository were linted on one Mac in one night, all sharing the default
cache (`~/Library/Caches/golangci-lint`). A clean branch made from main then reported three
`forbidigo` issues. The paths named `../../dotfiles-wt-goos/cli/internal/doctor/…`, a worktree
that had already been merged and removed. Earlier the same night, a run in another worktree
warned that it could not parse a file in `dotfiles-wt-ghstderr`, which was also gone.

## The Trap
The cache is keyed by package content, not by checkout. When two worktrees hold a package with
the same key, a run in one can be served results recorded in the other, still carrying that
worktree's paths. Those results may come from an intermediate state, such as a mutation run
with a `nolint` removed. So the cache can report issues the tree does not have, and by the same
mechanism it can hide issues the tree does have. A local "0 issues" is evidence about the cache
as much as about the code.

## The Solution
When lint names a path outside the tree being linted, or disagrees with what the code shows,
re-run with a fresh cache before believing it:
`GOLANGCI_LINT_CACHE=$(mktemp -d) golangci-lint run ./...`. Here that gave 0 issues. CI runs with
a fresh cache and is the authoritative answer. Locally, the fresh-cache run is the check to use
before a claim rests on a lint result, and especially after mutation runs in a sibling worktree.
