---
id: "lesson-563-a-squash-merge-closes-what-its-commits-say-not-what-the-body-says"
type: lesson
status: active
title: "A squash merge closes what its commits say, not what the body says"
created: "2026-11-06"
---

# A squash merge closes what its commits say, not what the body says

## Context

#3233 opened with `Closes #2445` in both its first commit message and its PR body. Triage then found that the fix
covered only half the issue: zoxide is installed only by the setup script's winget loop, which the PR left alone.
The PR body was edited to `Refs #2445`, and the rest was ticketed as #3238.

## Problem

The squash merge closed #2445 anyway. A squash commit's message is built from the messages of the commits it folds,
and the first commit still said `Closes #2445`. GitHub reads closing keywords from the merged commit as well as from
the PR body, so editing the body did not withdraw the promise. Nothing failed: the issue went from open to closed
with a linked merge, the normal outcome for a fixed bug.

## Solution

When a PR is narrowed from `Closes` to `Refs`, check its commit messages too. Then reopen whatever the merge closed,
with a comment naming what is still open. Better still, write `Refs` in the commit message until the PR's scope is
settled. A commit message cannot be edited without a force-push, while a body can be edited at any time.
