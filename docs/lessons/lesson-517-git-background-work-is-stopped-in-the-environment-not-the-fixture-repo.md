---
id: "lesson-517-git-background-work-is-stopped-in-the-environment-not-the-fixture-repo"
type: lesson
status: active
title: "Git background work is stopped in the environment, not in the fixture repo"
created: "2026-10-22"
---

# Git background work is stopped in the environment, not in the fixture repo

## Context

Go tests under `cli/internal` build throwaway repositories in `t.TempDir()` and run `git commit`, `clone`, `fetch`
and `push` against them. CI failed intermittently with `.git: directory not empty` during the TempDir cleanup, and once
with a `git gc --prune=now` that pruned nothing (#2803). #2802 had fixed one fixture by setting `gc.auto=0` and
`maintenance.auto=false` in that repository's config.

## Problem

Those git commands can fork a detached `gc --auto` or `maintenance run --auto` that keeps writing after the test
returns. Setting the keys in the fixture repo's config covers only the git commands run in that repository. A clone
does not inherit its remote's config, and a push into a bare remote runs `gc --auto` inside `receive-pack`, in the
remote. Most of the git in these packages is started by the code under test (a claim pushes and fetches, converge
clones), in repositories no fixture helper configured.

## Solution

`internal/gittest.Quiet()` sets `GIT_CONFIG_COUNT` with `gc.auto=0` and `maintenance.auto=false` in the test process's
environment. Each package whose tests run git calls it from `TestMain`, so every git the process starts sees the
keys: the fixture's, the code under test's, and the remote's `receive-pack`. `Quiet` accepts its own keys when a test
binary re-executes itself, which converge's helper-process tests do, and panics on any other `GIT_CONFIG_COUNT`.

## Rule

To change how git behaves for a whole test, set it in the environment the test process passes on, not in the
repositories it happens to create. The repo-config fix only covers the repositories the test creates, and gc can run in the ones it does not.
