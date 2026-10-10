---
id: "lesson-386-what-the-checkout-deleted-is-a-leftover-what-it-never-had-is-not"
type: lesson
status: active
title: "What the checkout deleted is a leftover; what it never had is not"
created: "2026-10-10"
---

# What the checkout deleted is a leftover; what it never had is not

## Context
Setup copies `.zsh/`, `ssh/` and `scripts/` from the checkout into `~/.dotfiles` with `cp -rf`, and
`dotf harness mirror` copies the same trees. Both only add. On msi, 2026-10-10,
`~/.dotfiles/scripts` held 45 scripts the repo had deleted. That directory is first on `PATH`, so
retired tools like `claude-mem-heal.sh` and `load-secrets.sh` still answered by name (#2266). #802
had counted 36 of them in August and fixed only `sensitive/` and `harness/`.

## The Trap
The obvious prune is "deploy dir minus checkout = garbage". #802 had already shown why that fails:
`sensitive/env-mapping.conf` is in the deploy dir, absent from the checkout, and it is
machine-local state. Deleting it breaks secret loading. Absence from the checkout says a file is
not ours today. It does not say the file was ever ours.

The opposite is also wrong. "Never prune, `doctor --fix` owns it" leaves the 45 scripts on `PATH`
until someone runs a command, and setup-windows already pruned retired scripts by a hand-kept name
list (WIN-013). That list had to grow with every retirement, and nobody grew it.

## The Solution
Provenance decides. A deploy-dir file absent from the checkout is pruned only when the checkout's
git history records deleting that path:
`git log --all --no-renames --diff-filter=D --name-only -- .zsh ssh scripts`. A file git never
tracked there is named and left. `harness.Mirror` prunes on every setup and `dotf converge`, on
every OS. `dotf doctor` reports a leftover on a box that has not mirrored since, and `--fix` removes
it through the same function. `sensitive/` and `secrets/` are outside the pruned trees.

The history is read only when there is an orphan to classify, so a converged deploy dir costs no git
call. Two cases prove nothing and prune nothing, and both are named: unreadable history (a tarball
checkout) and a shallow clone. A shallow clone's log is missing the deleting commits, so its empty
answer would read as "never ours".

## Takeaways
- **Git history is the provenance marker for copies of a checkout.** It is the same idea as the
  skill deploy's `generated: true`: delete only what you can prove you wrote. No hand-kept retired
  list is needed.
- **`--no-renames` matters.** Without it a renamed file's old path is reported as R, not D, and the
  leftover escapes.
- **A shallow clone is "unknown", not "clean".** Any check whose evidence is git history must ask
  `git rev-parse --is-shallow-repository` first. CI checks out at depth 1.
- **In a linked worktree `.git` is a file.** `isDir(".git")` skipped doctor's whole repo-to-deploy-dir
  drift section in every worktree. Use an existence check.
