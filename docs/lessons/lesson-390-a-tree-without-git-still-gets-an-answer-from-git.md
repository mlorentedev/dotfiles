---
id: "lesson-390-a-tree-without-git-still-gets-an-answer-from-git"
type: lesson
status: active
title: "A tree without .git still gets an answer from git"
created: "2026-10-10"
---

# A tree without .git still gets an answer from git

## Context
`dotf harness mirror` asks git which files the checkout ignores (`git ls-files --others --ignored
--exclude-standard`) and skips them, so a local or generated file never reaches `~/.dotfiles`
(#2268). A checkout with no `.git`, such as a tarball, has no ignore rules, so the mirror copies
everything, as it always did.

## The Trap
The first version decided "no `.git`" only after git failed. But git does not fail in a tree
without `.git`: it walks up to the nearest enclosing repository and answers for that one. A
tarball extracted under a home directory kept in git, with a catch-all `*` in its `.gitignore`,
got every file reported as ignored. The mirror skipped all of them and reported nothing.

The two git commands also disagree about paths. `ls-files` prints paths relative to the cwd, so
the enclosing repository's answer matched the deploy-relative paths exactly. `log --name-only`
prints paths relative to the repository root, so the history scan (`deletedPaths`) got
`tb/scripts/x`, matched nothing, and was harmless only by accident.

The unit test missed it twice. Its fake runner returned an error whether or not it was called.
The end-to-end test set `GIT_CEILING_DIRECTORIES`, which fenced git off from the very scenario.
Production never sets that variable. Found by PR-Agent's incremental review on #2277.

## The Solution
- `IgnoredInCheckout` checks `.git` with `Lstat` before it calls git. `Lstat` matches both a
  directory and the `.git` file of a linked worktree or submodule.
- The end-to-end test puts the checkout inside a repository whose `.gitignore` is `*`, and clears
  `GIT_CEILING_DIRECTORIES`.
- The unit test's runner fails the test if called, and would answer with a listing. Reverting
  the order fails both tests.

## Takeaways
- Decide whether a directory is a repository by looking for `.git`, never by whether a git
  command fails. git answers for the nearest repository above, not for the directory you named.
- A path from git is relative to the cwd or to the repository root depending on the command.
  Check which before you compare it with anything.
- A test that fences off the environment, with a ceiling or a stub that always errors, cannot
  see a bug that lives in the environment.
