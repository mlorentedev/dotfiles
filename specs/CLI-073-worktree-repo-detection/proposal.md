---
id: "CLI-073-worktree-repo-detection"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-09-29"
issue: "mlorentedev/dotfiles#1358"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-073: Recognize linked worktrees as checkouts

## Why

<!-- from issue #1358: CLI-073: env.RepoDir and doctor do not recognise a git worktree as a checkout (.git is a file there), so worktree-based work falls back to DOTFILES_REPO_DIR -->

`env.RepoDir` now recognizes a linked worktree's `.git` pointer file, but
`dotf doctor` still validates `DOTFILES_REPO_DIR` with `isDir(<repo>/.git)`.
Every branch-isolated validation therefore reports a false failure even though
Git itself recognizes the checkout.

## What

The repo-dir doctor check asks Git whether the resolved path is inside a work
tree, using the existing injected command seam. Normal clones and linked
worktrees pass; existing directories that are not Git checkouts still fail.

## Out of scope

- Changing `env.RepoDir`, whose worktree precedence already has regression tests.
- Auditing every historical `.git` directory assumption in unrelated checks.
- Treating arbitrary `.git` files as valid without confirmation from Git.

## Risks / open questions

- The check deliberately validates the configured cascade path rather than
  falling back to doctor's current working directory.
- Tests must use the existing `System.CommandOutput` seam; invoking the real Git
  binary in a unit test would make the result depend on the host.

## Acceptance criteria

- [x] A resolved linked worktree path passes the repo-dir doctor check.
- [x] A normal checkout continues to pass.
- [x] A missing path or an existing non-checkout continues to fail with the
  current diagnostic class.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related lesson: `docs/lessons/lesson-161-a-linked-worktree-s-checkout-is-not-self-contained.md`
- Related implementation: `cli/internal/env/env.go` (`RepoDir`) and
  `cli/internal/doctor/hookprobe.go` (`isGitCheckout`)
