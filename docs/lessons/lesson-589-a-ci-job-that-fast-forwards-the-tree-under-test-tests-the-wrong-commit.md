---
id: "lesson-589-a-ci-job-that-fast-forwards-the-tree-under-test-tests-the-wrong-commit"
type: lesson
status: active
title: "A CI job that fast-forwards the tree under test tests the wrong commit"
created: "2026-11-12"
---

# A CI job that fast-forwards the tree under test tests the wrong commit

## Context

`from-zero` runs `install.sh` on a fresh runner, then a second `dotf converge` that must change
nothing. Converge's first step, `checkout`, keeps the dotfiles checkout level with its upstream by
the `dotf update` rule: a clean branch behind `origin/main` is fast-forwarded, a detached or
upstream-less checkout is left alone and reported skipped.

## Problem

On a push to `main` the workflow's checkout is a `main` branch tracking `origin/main`, so the
first converge fast-forwarded it to whatever `main` held at that second, and the second converge
fast-forwarded again when another PR had landed in between. The no-op gate went red on both
runners with one changed entry, `checkout: fast-forwarded to origin/main`, twice in one morning
(`4b58ea1c`, `ec724137`), for a reason the diff could not contain. The quieter half: a passing run
did not prove the pushed commit either, since the first converge had already moved past it. Pull
requests never showed it, because their merge ref is checked out detached and the step skips.

## Solution

Check out the commit under test detached on every event that is not a pull request
(`ref: github.sha`), so the `checkout` step finds no upstream and skips on `main` as it does on a
pull request. Both converges then read one tree, and a green run names the commit it proved. The
rule generalises: a job that measures a tree must hold that tree still for the whole run, and a
step that is right on a workstation (follow the upstream) is a moving part in CI.

## Related

- #3375, the measurement; #3331, the same gate failing on a tool release that crossed
  `minimum_release_age` between the two converges.
- ADR-019 decision 2, the fast-forward rule the step applies.
