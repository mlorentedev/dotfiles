---
id: "lesson-591-git-archive-drops-what-export-ignore-names"
type: lesson
status: active
title: "git archive drops what export-ignore names, .github included"
created: "2026-11-12"
---

# git archive drops what export-ignore names, .github included

## Context

Filing the fleet-adoption issues for CI-018 meant running `dotf lint fleet` against each repository's
default branch tip, not its local checkout: the checkouts were up to 527 commits behind, and one sat
on a feature branch (#3384). The quick way to get a commit's `.github/` tree onto disk looked like
`git archive origin/main .github | tar -x`.

## Problem

One repository came back with no workflow files and `[OK] no fleet lint findings`. Its
`.gitattributes` says `.github export-ignore`, so `git archive` left the directory out and exited 0:
the linter saw an empty tree and reported it clean. A first attempt that also asked for `.gitea`
failed differently, because `git archive` errors on a path the commit lacks, and the pipe's `tar`
then extracted nothing for every repository, again reported as clean.

## Solution

Read a commit's files with `git ls-tree -r --name-only <rev> -- <paths>` and `git show <rev>:<file>`,
which ignore attributes, and count the files extracted before believing a clean result. `git archive`
is a release-tarball tool: it applies `export-ignore` and `export-subst` by design, so it is the wrong
reader for "what does this commit hold". The general rule is the one in `.claude/CLAUDE.md`'s
shell table: a step whose failure path reports "found nothing" needs a preflight that the input exists.

## Related

- #3384, `--fleet` reading the working tree instead of the default branch; its option (A) must not
  read files through `git archive`.
- #3147 (CI-018), the adoption run that measured this.
