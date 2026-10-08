---
id: "lesson-365-a-lint-finding-outside-the-worktree-is-the-cache-talking"
type: lesson
status: active
title: "A lint finding outside the worktree is the cache talking"
created: "2026-10-08"
---

# A lint finding outside the worktree is the cache talking

## Context
Parallel work in this repo happens in sibling worktrees (`../dotfiles-wt-*`), and every one of them
runs the same pinned `golangci-lint` against the same module path. The linter keeps one cache per
user, keyed by package content rather than by checkout.

## The Trap
On 2026-10-08, `golangci-lint run ./...` in `dotfiles-wt-hr` reported two `forbidigo` findings in
`../../dotfiles-wt-dm/cli/internal/doctor/fs.go` and `system.go`. `dotfiles-wt-dm` had already been
deleted when its PR merged. An earlier run in the same session had warned that it could not parse a
file in `dotfiles-wt-goos`, a worktree that belonged to another session. Neither finding was about
the code under test. Fixing one would have meant editing a file that no longer existed, and
dismissing it would have taught the habit of dismissing lint output. Either way, `N issues` was no
longer a statement about this worktree.

## The Solution
When a finding's path is outside the current worktree, run `golangci-lint cache clean` and lint
again. After the clean run, the same tree reported `0 issues`. The rule is now in
`.claude/CLAUDE.md` next to the pinned-linter rule (BUG-071), which is the same failure from the
other direction: local lint output that does not describe what CI will see.

A tool's verdict covers only what it actually read. When its output names a file you did not give
it, it is answering for another input.
