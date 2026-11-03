---
id: "lesson-547-actionlint-with-no-arguments-lints-only-github-workflows"
type: lesson
status: active
title: "actionlint with no arguments lints only .github/workflows"
created: "2026-11-02"
---

# actionlint with no arguments lints only .github/workflows

## Context

CLI-133 (#2965) makes `dotf init` emit a `lint.yml` for every stack, copied from `mlorentedev/ci-workflows`.
That workflow runs `actionlint` with no arguments, which on GitHub lints every file under `.github/workflows/`.
`dotf init` writes workflows to the forge's directory, so a Gitea or Forgejo repository gets
`.gitea/workflows/lint.yml` or `.forgejo/workflows/lint.yml`.

## Problem

With no arguments, actionlint looks for `.github/workflows/` and nothing else. A copied workflow with that
default would lint nothing in a Gitea repository, or stop because it finds no project. Either way the workflow
it was added to check would go unchecked. Two other checks in the same change looked fine but checked the wrong
thing. `scripts/check-actions-pinned.sh <dir>` reported `OK (15 workflow files)` on a scaffolded repository with
one workflow: the script takes its root from its own location and ignores the argument, so it had checked this
repository.

## Solution

The template names the directory (`actionlint … .github/workflows/*.y*ml`), and `dotf init` replaces it with the
forge's workflow directory. A test asserts the Gitea case. The pin check ran from a copy placed inside the
scaffolded repository, with a negative control: an unpinned `uses:` in that copy exits 1. The general rule is that
a count that does not match the tree you meant to check shows the check ran somewhere else. Read the count, not
only `OK`.
